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
	"strconv"

	"net/http"
        "encoding/json"
        "bytes"
        "io/ioutil"

	"github.com/daltonhahn/anvil/acl"
)

const DebugCM = 1

type LogEntry struct {
	ACLObj	acl.ACLEntry
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
	commitIndex	int
	lastApplied	int
	state              CMState
	electionResetEvent time.Time
	nextIndex	map[int]int
	matchIndex	map[int]int
}


var CM ConsensusModule

func NewConsensusModule(id string, peerIds []string) *ConsensusModule {
	CM := new(ConsensusModule)
	CM.id = id
	CM.PeerIds = peerIds
	CM.state = Follower
	CM.votedFor = ""
	CM.nextIndex = make(map[int]int)
	CM.matchIndex = make(map[int]int)
	CM.commitIndex = -1
	CM.lastApplied = -1

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

func GetLog() {
	for _, ele := range CM.log {
		fmt.Println(ele.ACLObj.Name)
	}
}

func TokenLookup(token string, targetSvc string, requestTime time.Time) bool {
	for _, ele := range CM.log {
		if ele.ACLObj.TokenValue == token {
			for _,svc := range ele.ACLObj.ServiceList {
				if svc == targetSvc && requestTime.Before(ele.ACLObj.ExpirationTime) {
					return true
				}
			}
		}
	}
	return false
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


//Pass this function any data type and it will return a boolean of whether it was appended to the log
// Change this function to send a REST API Request to Leader instead
// Will require the creation of a function within the catalog to return the current leader of the cluster
func Submit(command acl.ACLEntry) bool {
	CM.mu.Lock()
	defer CM.mu.Unlock()

	dlog(fmt.Sprintf("Submit received by %v: %v", CM.state, command))
	if CM.state == Leader {
		CM.log = append(CM.log, LogEntry{ACLObj: command, Term: CM.currentTerm})
		dlog(fmt.Sprintf("... log=%v", CM.log))
		return true
	}
	return false
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
	lastLogIndex, lastLogTerm := lastLogIndexAndTerm()
	dlog(fmt.Sprintf("RequestVote: %+v [currentTerm=%d, votedFor=%d log index/term=(%d, %d)]", args, CM.currentTerm, CM.votedFor, lastLogIndex, lastLogTerm))

	if args.Term > CM.currentTerm {
		dlog("... term out of date in RequestVote")
		leader := getLeader(CM.id)
		UpdateLeader(CM.id, leader)
		becomeFollower(args.Term)
	}

	if CM.currentTerm == args.Term &&
		(CM.votedFor == "" || CM.votedFor == args.CandidateId) &&
		(args.LastLogTerm > lastLogTerm ||
		  (args.LastLogTerm == lastLogTerm && args.LastLogIndex >= lastLogIndex)) {
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
		UpdateLeader(CM.id, args.LeaderId)
		becomeFollower(args.Term)
	}

	reply.Success = false
	if args.Term == CM.currentTerm {
		if CM.state != Follower {
			UpdateLeader(CM.id, args.LeaderId)
			becomeFollower(args.Term)
		}
		CM.electionResetEvent = time.Now()
		// Does our log contain an entry at PrevLogIndex whose term matches
		// PrevLogTerm? Note that in the extreme case of PrevLogIndex=-1 this is
		// vacuously true.
		// If your log has fewer entries than the Leader, then you need to update your log
		if args.PrevLogIndex > len(CM.log) {
			_, backlogEntries := BacklogRequest(args.LeaderId)
			//Figure out a way to add these into your log and update the relevant values
			dlog(fmt.Sprintf("Fast forwarding backlog"))
                        CM.log = append(CM.log[:CM.commitIndex], backlogEntries...)
			CM.commitIndex = CM.commitIndex + len(backlogEntries)
		}
		if args.PrevLogIndex == -1 || (args.PrevLogIndex < len(CM.log) && args.PrevLogTerm == CM.log[args.PrevLogIndex].Term) {
			reply.Success = true

			// Find an insertion point - where there's a term mismatch between
			// the existing log starting at PrevLogIndex+1 and the new entries sent
			// in the RPC.
			logInsertIndex := args.PrevLogIndex + 1
			newEntriesIndex := 0

			for {
				if logInsertIndex >= len(CM.log) || newEntriesIndex >= len(args.Entries) {
					break
				}
				if CM.log[logInsertIndex].Term != args.Entries[newEntriesIndex].Term {
					break
				}
				logInsertIndex++
				newEntriesIndex++
			}
			// At the end of this loop:
			// - logInsertIndex points at the end of the log, or an index where the
			//   term mismatches with an entry from the leader
			// - newEntriesIndex points at the end of Entries, or an index where the
			//   term mismatches with the corresponding log entry
			if newEntriesIndex < len(args.Entries) {
				dlog(fmt.Sprintf("... inserting entries %v from index %d", args.Entries[newEntriesIndex:], logInsertIndex))
				CM.log = append(CM.log[:logInsertIndex], args.Entries[newEntriesIndex:]...)
				dlog(fmt.Sprintf("... log is now: %v", CM.log))
			}

			// Set commit index.
			if args.LeaderCommit > CM.commitIndex {
				CM.commitIndex = intMin(args.LeaderCommit, len(CM.log)-1)
				dlog(fmt.Sprintf("... setting commitIndex=%d", CM.commitIndex))
			}
		}
		reply.Success = true
	}
	reply.Term = CM.currentTerm
	return reply
}


func electionTimeout() time.Duration {
	duration := 2000
	if len(os.Getenv("RAFT_FORCE_MORE_REELECTION")) > 0 && rand.Intn(3) == 0 {
		return time.Duration(duration) * time.Millisecond
	} else {
		return time.Duration(duration+rand.Intn(150)) * time.Millisecond
	}
}

func runElectionTimer(myid string) {
	timeoutDuration := electionTimeout()
	CM.mu.Lock()
	termStarted := CM.currentTerm
	CM.mu.Unlock()
	dlog(fmt.Sprintf("election timer started (%v), term=%d", timeoutDuration, termStarted))
	CM.id = myid

	ticker := time.NewTicker(1000 * time.Millisecond)
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

		if elapsed := time.Since(CM.electionResetEvent); elapsed >= timeoutDuration {
			startElection()
			CM.mu.Unlock()
			return
		}
		CM.mu.Unlock()
	}
}

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
		for _, peerId := range CM.PeerIds {
			go func(peerId string) {
				CM.mu.Lock()
				savedLastLogIndex, savedLastLogTerm := lastLogIndexAndTerm()
				CM.mu.Unlock()

				args := RequestVoteArgs{
					Term:         savedCurrentTerm,
					CandidateId:  CM.id,
					LastLogIndex: savedLastLogIndex,
					LastLogTerm:  savedLastLogTerm,
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
						leader := getLeader(CM.id)
						UpdateLeader(CM.id, leader)
						becomeFollower(reply.Term)
						return
					} else if reply.Term == savedCurrentTerm {
						if reply.VoteGranted {
							votes := int(atomic.AddInt32(&votesReceived, 1))
							if votes*2 > len(CM.PeerIds)+1 {
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
	go runElectionTimer(CM.id)
}

func becomeFollower(term int) {
	dlog(fmt.Sprintf("becomes Follower with term=%d; log=%v", term, CM.log))
	CM.state = Follower
	CM.currentTerm = term
	CM.votedFor = ""
	CM.electionResetEvent = time.Now()

	go runElectionTimer(CM.id)
}

func startLeader() {
	CM.state = Leader
	dlog(fmt.Sprintf("becomes Leader; term=%d, log=%v", CM.currentTerm, CM.log))

	UpdateLeader(CM.id, CM.id)
	if len(CM.PeerIds) == 0 {
        } else {
		for ind, ele := range CM.PeerIds {
			UpdateLeader(ele, CM.id)
			CM.nextIndex[ind] = len(CM.log)
			CM.matchIndex[ind] = -1
		}
	}

	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()

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

func leaderSendHeartbeats() {
	CM.mu.Lock()
	savedCurrentTerm := CM.currentTerm
	CM.mu.Unlock()
	if CM.nextIndex == nil {
		CM.nextIndex = make(map[int]int)
	}
	if CM.matchIndex == nil {
		CM.matchIndex = make(map[int]int)
	}
	var ni int

        if len(CM.PeerIds) == 0 {
                return
        } else {
		  for ind, peerId := range CM.PeerIds {
			go func(peerId string) {
				if peerId == "" {
					return
				}
				CM.mu.Lock()
				ni = CM.nextIndex[ind]
				prevLogIndex := ni - 1
				prevLogTerm := -1
				if prevLogIndex >= 0 {
					prevLogTerm = CM.log[prevLogIndex].Term
				}
				entries := CM.log[ni:]

				args := AppendEntriesArgs{
					Term:         savedCurrentTerm,
					LeaderId:     CM.id,
					PrevLogIndex: prevLogIndex,
					PrevLogTerm:  prevLogTerm,
					Entries:      entries,
					LeaderCommit: CM.commitIndex,
				}
				CM.mu.Unlock()
				var reply AppendEntriesReply
				err, reply := SendAppendEntry(peerId, args)
				if err == nil {
					CM.mu.Lock()
					defer CM.mu.Unlock()
					if (reply.Term > savedCurrentTerm || reply.Term == 0 || savedCurrentTerm == 0) {
						dlog(fmt.Sprintf("term out of date in heartbeat reply"))
						leader := getLeader(CM.id)
						UpdateLeader(CM.id, leader)
						becomeFollower(reply.Term)
						return
					}
				}

				if CM.state == Leader && savedCurrentTerm == reply.Term {
					if reply.Success {
						CM.nextIndex[ind] = ni + len(entries)
						CM.matchIndex[ind] = CM.nextIndex[ind] - 1

						savedCommitIndex := CM.commitIndex
						for i := CM.commitIndex + 1; i < len(CM.log); i++ {
							if CM.log[i].Term == CM.currentTerm {
								matchCount := 1
								for ind, _ := range CM.PeerIds {
									if CM.matchIndex[ind] >= i {
										matchCount++
									}
								}
								if matchCount*2 > len(CM.PeerIds)+1 {
									CM.commitIndex = i
								}
							}
						}
						if CM.commitIndex != savedCommitIndex {
							dlog(fmt.Sprintf("leader sets commitIndex := %d", CM.commitIndex))
						}
					} else {
						CM.nextIndex[ind] = ni - 1
						dlog(fmt.Sprintf("AppendEntries reply from %d !success: nextIndex := %d", peerId, ni-1))
					}
				}
			}(peerId)
		}
	}
}

func getLeader(target string) (string) {
        resp, err := http.Get("http://" + target + ":443/anvil/catalog/leader")
        if err != nil {
                return ""
        }

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
                return ""
        }
	defer resp.Body.Close()
	return string(body)
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

func BacklogRequest(leader string) (error, []LogEntry) {
	resp, err := http.Get("http://" + leader + ":443/anvil/raft/backlog/" + strconv.Itoa(CM.commitIndex))
        if err != nil {
		return errors.New("No HTTP response"), []LogEntry{}
        }

        body, err := ioutil.ReadAll(resp.Body)
        if err != nil {
		return errors.New("Bad Read Error"), []LogEntry{}
        }
        var newEntries []LogEntry

        err = json.Unmarshal(body, &newEntries)
        if err != nil {
		return errors.New("JSON Parse Error"), []LogEntry{}
        }
	return nil, newEntries
}

func PullBacklogEntries(index int64) []LogEntry {
	backlog := CM.log[index:]
	return backlog
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

func intMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func lastLogIndexAndTerm() (int, int) {
	if len(CM.log) > 0 {
		lastIndex := len(CM.log) - 1
		return lastIndex, CM.log[lastIndex].Term
	} else {
		return -1, -1
	}
}
