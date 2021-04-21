package raft

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
	"time"
	"errors"

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
	CM := new(ConsensusModule)
	CM.id = id
	CM.PeerIds = peerIds
	CM.state = Follower
	CM.votedFor = ""

	go func() {
		CM.mu.Lock()
		CM.electionResetEvent = time.Now()
		CM.mu.Unlock()
		runElectionTimer(CM.id)
	}()

	return CM
}

func GetPeers() {
	for _, ele := range CM.PeerIds {
		fmt.Println(ele)
	}
}

func Report() (id string, term int, isLeader bool) {
	CM.mu.Lock()
	defer CM.mu.Unlock()
	return CM.id, CM.currentTerm, CM.state == Leader
}

func Stop() {
	CM.mu.Lock()
	defer CM.mu.Unlock()
	CM.state = Dead
	dlog("becomes Dead")
}

func dlog(format string) {
	if DebugCM > 0 {
		format = fmt.Sprintf("[%d] ", CM.id) + format
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

func RequestVote(args RequestVoteArgs) RequestVoteReply {
	CM.mu.Lock()
	defer CM.mu.Unlock()
	reply := RequestVoteReply{}
	if CM.state == Dead {
		return RequestVoteReply{CM.currentTerm, false}
	}
	dlog(fmt.Sprintf("RequestVote: %+v [currentTerm=%d, votedFor=%d]", args, CM.currentTerm, CM.votedFor))

	if args.Term > CM.currentTerm {
		dlog("... term out of date in RequestVote")
		becomeFollower(args.Term)
	}

	if CM.currentTerm == args.Term &&
		(CM.votedFor == "" || CM.votedFor == args.CandidateId) {
		reply.Term = CM.currentTerm
		reply.VoteGranted = true
		CM.votedFor = args.CandidateId
		CM.electionResetEvent = time.Now()
	} else {
		reply.Term = CM.currentTerm
		reply.VoteGranted = false
	}
	reply.Term = CM.currentTerm
	dlog(fmt.Sprintf("... RequestVote reply: %+v", reply))
	return reply
}

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

func AppendEntries(args AppendEntriesArgs) AppendEntriesReply {
	CM.mu.Lock()
	defer CM.mu.Unlock()
	reply := AppendEntriesReply{}
	if CM.state == Dead {
		return AppendEntriesReply{CM.currentTerm, false}
	}

	if args.Term > CM.currentTerm {
		dlog("... term out of date in AppendEntries")
		becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == CM.currentTerm {
		if CM.state != Follower {
			becomeFollower(args.Term)
		}
		CM.electionResetEvent = time.Now()
		reply.Success = true
	}

	reply.Term = CM.currentTerm
	return reply
}


// electionTimeout generates a pseudo-random election timeout duration.
func electionTimeout() time.Duration {
	//TUNE ME FOR TESTING
	duration := 2000
	// If RAFT_FORCE_MORE_REELECTION is set, stress-test by deliberately
	// generating a hard-coded number very often. This will create collisions
	// between different servers and force more re-elections.
	if len(os.Getenv("RAFT_FORCE_MORE_REELECTION")) > 0 && rand.Intn(3) == 0 {
		return time.Duration(duration) * time.Millisecond
	} else {
		return time.Duration(duration+rand.Intn(150)) * time.Millisecond
	}
}

// runElectionTimer implements an election timer. It should be launched whenever
// we want to start a timer towards becoming a candidate in a new election.
//
// This function is blocking and should be launched in a separate goroutine;
// it's designed to work for a single (one-shot) election timer, as it exits
// whenever the CM state changes from follower/candidate or the term changes.
func runElectionTimer(myid string) {
	timeoutDuration := electionTimeout()
	CM.mu.Lock()
	termStarted := CM.currentTerm
	CM.mu.Unlock()
	dlog(fmt.Sprintf("election timer started (%v), term=%d", timeoutDuration, termStarted))
	CM.id = myid

	// This loops until either:
	// - we discover the election timer is no longer needed, or
	// - the election timer expires and this CM becomes a candidate
	// In a follower, this typically keeps running in the background for the
	// duration of the CM's lifetime.
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		<-ticker.C

		CM.mu.Lock()
		if CM.state != Candidate && CM.state != Follower {
			dlog(fmt.Sprintf("in election timer state=%s, bailing out", CM.state))
			CM.mu.Unlock()
			return
		}

		if termStarted != CM.currentTerm {
			dlog(fmt.Sprintf("in election timer term changed from %d to %d, bailing out", termStarted, CM.currentTerm))
			CM.mu.Unlock()
			return
		}

		// Start an election if we haven't heard from a leader or haven't voted for
		// someone for the duration of the timeout.
		if elapsed := time.Since(CM.electionResetEvent); elapsed >= timeoutDuration {
			startElection()
			CM.mu.Unlock()
			return
		}
		CM.mu.Unlock()
	}
}

// startElection starts a new election with this CM as a candidate.
// Expects cm.mu to be locked.
func startElection() {
	CM.state = Candidate
	CM.currentTerm += 1
	savedCurrentTerm := CM.currentTerm
	CM.electionResetEvent = time.Now()
	CM.votedFor = CM.id
	dlog(fmt.Sprintf("becomes Candidate (currentTerm=%d); log=%v", savedCurrentTerm, CM.log))

	var votesReceived int32 = 1

	if len(CM.PeerIds) == 0 {
		startLeader()
		return
	} else {
		// Send RequestVote RPCs to all other servers concurrently.
		for _, peerId := range CM.PeerIds {
			go func(peerId string) {

				args := RequestVoteArgs{
					Term:        savedCurrentTerm,
					CandidateId: CM.id,
				}
				var reply RequestVoteReply

				dlog(fmt.Sprintf("sending RequestVote to %d: %+v", peerId, args))

				err, reply := SendVoteReq(peerId, args)
				if err == nil {
					CM.mu.Lock()
					defer CM.mu.Unlock()
					dlog(fmt.Sprintf("received RequestVoteReply %+v", reply))

					if CM.state != Candidate {
						dlog(fmt.Sprintf("while waiting for reply, state = %v", CM.state))
						return
					}

					if reply.Term > savedCurrentTerm {
						dlog("term out of date in RequestVoteReply")
						becomeFollower(reply.Term)
						return
					} else if reply.Term == savedCurrentTerm {
						if reply.VoteGranted {
							votes := int(atomic.AddInt32(&votesReceived, 1))
							if votes*2 > len(CM.PeerIds)+1 {
								// Won the election!
								dlog(fmt.Sprintf("wins election with %d votes", votes))
								startLeader()
								return
							}
						}
					}
				}
			}(peerId)
		}
	}
	// Run another election timer, in case this election is not successful.
	go runElectionTimer(CM.id)
}

// becomeFollower makes cm a follower and resets its state.
// Expects cm.mu to be locked.
func becomeFollower(term int) {
	dlog(fmt.Sprintf("becomes Follower with term=%d; log=%v", term, CM.log))
	CM.state = Follower
	CM.currentTerm = term
	CM.votedFor = ""
	CM.electionResetEvent = time.Now()

	go runElectionTimer(CM.id)
}

// startLeader switches cm into a leader state and begins process of heartbeats.
// Expects cm.mu to be locked.
func startLeader() {
	CM.state = Leader
	dlog(fmt.Sprintf("becomes Leader; term=%d, log=%v", CM.currentTerm, CM.log))

	UpdateLeader(CM.id, CM.id)
	if len(CM.PeerIds) == 0 {
        } else {
		for _, ele := range CM.PeerIds {
			UpdateLeader(ele, CM.id)
		}
	}

	go func() {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		// Send periodic heartbeats, as long as still leader.
		for {
			leaderSendHeartbeats()
			<-ticker.C

			CM.mu.Lock()
			if CM.state != Leader {
				CM.mu.Unlock()
				return
			}
			CM.mu.Unlock()
		}
	}()
}

// leaderSendHeartbeats sends a round of heartbeats to all peers, collects their
// replies and adjusts cm's state.
func leaderSendHeartbeats() {
	CM.mu.Lock()
	savedCurrentTerm := CM.currentTerm
	CM.mu.Unlock()

        if len(CM.PeerIds) == 0 {
                return
        } else {
		for _, peerId := range CM.PeerIds {
			args := AppendEntriesArgs{
				Term:     savedCurrentTerm,
				LeaderId: CM.id,
			}
			go func(peerId string) {
				if peerId == "" {
					return
				}
				UpdateLeader(peerId, CM.id)

				var reply AppendEntriesReply
				err, reply := SendAppendEntry(peerId, args)
				if err == nil {
					CM.mu.Lock()
					defer CM.mu.Unlock()
					if reply.Term > savedCurrentTerm {
						dlog("term out of date in heartbeat reply")
						becomeFollower(reply.Term)
						return
					}
				}
			}(peerId)
		}
	}
}

func UpdateLeader(target string, newLeader string) {
	reqBody, _ := json.Marshal(map[string]string {
		"leader": newLeader,
	})

	postBody := bytes.NewBuffer(reqBody)

	resp, err := http.Post("http://" + target + ":443/anvil/raft/updateleader", "application/json", postBody)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return
}

func SendAppendEntry(target string, args AppendEntriesArgs) (error, AppendEntriesReply) {
        reqBody, _ := json.Marshal(args)
        postBody := bytes.NewBuffer(reqBody)
	resp, err := http.Post("http://" + target + ":443/anvil/raft/appendentries", "application/json", postBody)
        if err != nil {
		return errors.New("No HTTP response"), AppendEntriesReply{}
        }
        defer resp.Body.Close()

        b, err := ioutil.ReadAll(resp.Body)
        if err != nil {
		return errors.New("Bad read error"), AppendEntriesReply{}
        }
        var ae_reply AppendEntriesReply
        err = json.Unmarshal(b, &ae_reply)
        if err != nil {
		return errors.New("JSON Parse error"), AppendEntriesReply{}
        }
        return nil, ae_reply
}


func SendVoteReq(target string, args RequestVoteArgs) (error, RequestVoteReply) {
        reqBody, _ := json.Marshal(args)
        postBody := bytes.NewBuffer(reqBody)
	resp, err := http.Post("http://" + target + ":443/anvil/raft/requestvote", "application/json", postBody)
        if err != nil {
		return errors.New("No HTTP response"), RequestVoteReply{}
        }
        defer resp.Body.Close()

        b, err := ioutil.ReadAll(resp.Body)
        if err != nil {
		return errors.New("Bad read error"), RequestVoteReply{}
        }
        var rv_reply RequestVoteReply
        err = json.Unmarshal(b, &rv_reply)
        if err != nil {
		return errors.New("JSON Parse error"), RequestVoteReply{}
        }
        return nil, rv_reply
}

