package raft

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"net/http"
        "encoding/json"
        "bytes"
        "io/ioutil"
)

const DebugCM = 1

type LogEntry struct {
	Command interface{}
	Term    int
}

type CMState int

const (
	Follower CMState = iota
	Candidate
	Leader
	Dead
)

func (s CMState) String() string {
	switch s {
	case Follower:
		return "Follower"
	case Candidate:
		return "Candidate"
	case Leader:
		return "Leader"
	case Dead:
		return "Dead"
	default:
		panic("unreachable")
	}
}

type ConsensusModule struct {
	mu sync.Mutex
	id string
	PeerIds []string
	currentTerm int
	votedFor    string
	log         []LogEntry
	state              CMState
	electionResetEvent time.Time
}


var CM ConsensusModule

// NewConsensusModule creates a new CM with the given ID, list of peer IDs and
// server. The ready channel signals the CM that all peers are connected and
// it's safe to start its state machine.
func NewConsensusModule(id string, peerIds []string) *ConsensusModule {
	cm := new(ConsensusModule)
	cm.id = id
	cm.PeerIds = peerIds
	cm.state = Follower
	cm.votedFor = ""

	go func() {
		// The CM is quiescent until ready is signaled; then, it starts a countdown
		// for leader election.
		//<-ready
		cm.mu.Lock()
		cm.electionResetEvent = time.Now()
		cm.mu.Unlock()
		cm.runElectionTimer()
	}()

	return cm
}

func (cm *ConsensusModule) Report() (id string, term int, isLeader bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.id, cm.currentTerm, cm.state == Leader
}

func (cm *ConsensusModule) Stop() {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.state = Dead
	cm.dlog("becomes Dead")
}

func (cm *ConsensusModule) dlog(format string) {
	if DebugCM > 0 {
		format = fmt.Sprintf("[%d] ", cm.id) + format
		log.Printf(format)
	}
}

type RequestVoteArgs struct {
	Term         int `json:"term"`
	CandidateId  string `json:"candidateid"`
	LastLogIndex int `json:"lastlogindex"`
	LastLogTerm  int `json:"lastlogterm"`
}

type RequestVoteReply struct {
	Term        int `json:"term"`
	VoteGranted bool `json:"votegranted"`
}

func (cm *ConsensusModule) RequestVote(args RequestVoteArgs) RequestVoteReply {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	reply := RequestVoteReply{}
	if cm.state == Dead {
		return RequestVoteReply{cm.currentTerm, false}
	}
	cm.dlog(fmt.Sprintf("RequestVote: %+v [currentTerm=%d, votedFor=%d]", args, cm.currentTerm, cm.votedFor))

	if args.Term > cm.currentTerm {
		cm.dlog("... term out of date in RequestVote")
		cm.becomeFollower(args.Term)
	}

	if cm.currentTerm == args.Term &&
		(cm.votedFor == "" || cm.votedFor == args.CandidateId) {
		reply.Term = cm.currentTerm
		reply.VoteGranted = true
		cm.votedFor = args.CandidateId
		cm.electionResetEvent = time.Now()
	} else {
		reply.Term = cm.currentTerm
		reply.VoteGranted = false
	}
	reply.Term = cm.currentTerm
	cm.dlog(fmt.Sprintf("... RequestVote reply: %+v", reply))
	return reply
}
/*

// See figure 2 in the paper.
type AppendEntriesArgs struct {
	Term     int
	LeaderId string

	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
}

func (cm *ConsensusModule) AppendEntries(args AppendEntriesArgs, reply *AppendEntriesReply) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	if cm.state == Dead {
		return nil
	}
	cm.dlog("AppendEntries: %+v", args)

	if args.Term > cm.currentTerm {
		cm.dlog("... term out of date in AppendEntries")
		cm.becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == cm.currentTerm {
		if cm.state != Follower {
			cm.becomeFollower(args.Term)
		}
		cm.electionResetEvent = time.Now()
		reply.Success = true
	}

	reply.Term = cm.currentTerm
	cm.dlog("AppendEntries reply: %+v", *reply)
	return nil
}

*/
// electionTimeout generates a pseudo-random election timeout duration.
func (cm *ConsensusModule) electionTimeout() time.Duration {
	// If RAFT_FORCE_MORE_REELECTION is set, stress-test by deliberately
	// generating a hard-coded number very often. This will create collisions
	// between different servers and force more re-elections.
	if len(os.Getenv("RAFT_FORCE_MORE_REELECTION")) > 0 && rand.Intn(3) == 0 {
		return time.Duration(150) * time.Millisecond
	} else {
		return time.Duration(150+rand.Intn(150)) * time.Millisecond
	}
}

// runElectionTimer implements an election timer. It should be launched whenever
// we want to start a timer towards becoming a candidate in a new election.
//
// This function is blocking and should be launched in a separate goroutine;
// it's designed to work for a single (one-shot) election timer, as it exits
// whenever the CM state changes from follower/candidate or the term changes.
func (cm *ConsensusModule) runElectionTimer() {
	timeoutDuration := cm.electionTimeout()
	cm.mu.Lock()
	termStarted := cm.currentTerm
	cm.mu.Unlock()
	cm.dlog(fmt.Sprintf("election timer started (%v), term=%d", timeoutDuration, termStarted))

	// This loops until either:
	// - we discover the election timer is no longer needed, or
	// - the election timer expires and this CM becomes a candidate
	// In a follower, this typically keeps running in the background for the
	// duration of the CM's lifetime.
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C

		cm.mu.Lock()
		if cm.state != Candidate && cm.state != Follower {
			cm.dlog(fmt.Sprintf("in election timer state=%s, bailing out", cm.state))
			cm.mu.Unlock()
			return
		}

		if termStarted != cm.currentTerm {
			cm.dlog(fmt.Sprintf("in election timer term changed from %d to %d, bailing out", termStarted, cm.currentTerm))
			cm.mu.Unlock()
			return
		}

		// Start an election if we haven't heard from a leader or haven't voted for
		// someone for the duration of the timeout.
		if elapsed := time.Since(cm.electionResetEvent); elapsed >= timeoutDuration {
			cm.startElection()
			cm.mu.Unlock()
			return
		}
		cm.mu.Unlock()
	}
}

// startElection starts a new election with this CM as a candidate.
// Expects cm.mu to be locked.
func (cm *ConsensusModule) startElection() {
	cm.state = Candidate
	cm.currentTerm += 1
	savedCurrentTerm := cm.currentTerm
	cm.electionResetEvent = time.Now()
	cm.votedFor = cm.id
	cm.dlog(fmt.Sprintf("becomes Candidate (currentTerm=%d); log=%v", savedCurrentTerm, cm.log))

	var votesReceived int32 = 1

	// Send RequestVote RPCs to all other servers concurrently.
	for _, peerId := range cm.PeerIds {
		go func(peerId string) {
			//Debugging no peers
			if peerId == "" {
				cm.startLeader()
				return
			}


			args := RequestVoteArgs{
				Term:        savedCurrentTerm,
				CandidateId: cm.id,
			}
			var reply RequestVoteReply

			cm.dlog(fmt.Sprintf("sending RequestVote to %d: %+v", peerId, args))

			err, reply := SendVoteReq(peerId, args)
			if err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				cm.dlog(fmt.Sprintf("received RequestVoteReply %+v", reply))

				if cm.state != Candidate {
					cm.dlog(fmt.Sprintf("while waiting for reply, state = %v", cm.state))
					return
				}

				if reply.Term > savedCurrentTerm {
					cm.dlog("term out of date in RequestVoteReply")
					cm.becomeFollower(reply.Term)
					return
				} else if reply.Term == savedCurrentTerm {
					if reply.VoteGranted {
						votes := int(atomic.AddInt32(&votesReceived, 1))
						if votes*2 > len(cm.PeerIds)+1 {
							// Won the election!
							cm.dlog(fmt.Sprintf("wins election with %d votes", votes))
							cm.startLeader()
							return
						}
					}
				}
			}
		}(peerId)
	}
	// Run another election timer, in case this election is not successful.
	go cm.runElectionTimer()
}

// becomeFollower makes cm a follower and resets its state.
// Expects cm.mu to be locked.
func (cm *ConsensusModule) becomeFollower(term int) {
	cm.dlog(fmt.Sprintf("becomes Follower with term=%d; log=%v", term, cm.log))
	cm.state = Follower
	cm.currentTerm = term
	cm.votedFor = ""
	cm.electionResetEvent = time.Now()

	go cm.runElectionTimer()
}

// startLeader switches cm into a leader state and begins process of heartbeats.
// Expects cm.mu to be locked.
func (cm *ConsensusModule) startLeader() {
	cm.state = Leader
	cm.dlog(fmt.Sprintf("becomes Leader; term=%d, log=%v", cm.currentTerm, cm.log))

	/*
	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		// Send periodic heartbeats, as long as still leader.
		for {
			cm.leaderSendHeartbeats()
			<-ticker.C

			cm.mu.Lock()
			if cm.state != Leader {
				cm.mu.Unlock()
				return
			}
			cm.mu.Unlock()
		}
	}()
	*/
}

/*
// leaderSendHeartbeats sends a round of heartbeats to all peers, collects their
// replies and adjusts cm's state.
func (cm *ConsensusModule) leaderSendHeartbeats() {
	cm.mu.Lock()
	savedCurrentTerm := cm.currentTerm
	cm.mu.Unlock()

	for _, peerId := range cm.PeerIds {
		args := AppendEntriesArgs{
			Term:     savedCurrentTerm,
			LeaderId: cm.id,
		}
		go func(peerId string) {
			cm.dlog("sending AppendEntries to %v: ni=%d, args=%+v", peerId, 0, args)
			var reply AppendEntriesReply
			// REPLACE WITH API CALL INSTEAD OF RPC
			//if err := cm.server.Call(peerId, "ConsensusModule.AppendEntries", args, &reply); err == nil {
				cm.mu.Lock()
				defer cm.mu.Unlock()
				if reply.Term > savedCurrentTerm {
					cm.dlog("term out of date in heartbeat reply")
					cm.becomeFollower(reply.Term)
					return
				}
			}
		}(peerId)
	}
}
*/


func SendVoteReq(target string, args RequestVoteArgs) (error, RequestVoteReply) {
        reqBody, _ := json.Marshal(args)
        postBody := bytes.NewBuffer(reqBody)
        resp, err := http.Post("http://" + target + "/anvil/raft/requestvote", "application/json", postBody)
        if err != nil {
                log.Fatalln("Unable to post content")
        }
        defer resp.Body.Close()

        b, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                log.Fatalln("Unable to read received content")
        }
        var rv_reply RequestVoteReply
        err = json.Unmarshal(b, &rv_reply)
        if err != nil {
                log.Fatalln("Unable to process response JSON")
        }
        return nil, rv_reply
}
