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
	"net"

	"net/http"
        "encoding/json"
        "bytes"
        "io/ioutil"

	"github.com/daltonhahn/anvil/acl"
	"github.com/daltonhahn/anvil/security"
	"gopkg.in/yaml.v2"
)

type AssignmentMap struct {
	Quorum		[]string	`yaml:"quorum,omitempty"`
        Nodes		[]string	`yaml:"nodes,omitempty"`
	SvcMap		[]ACLMap	`yaml:"svcmap,omitempty"`
	Gossip		bool
	Iteration	int
	Prefix		string
}

type ACLMap struct {
        Node            string
        Svc             string
        TokName         string
	Valid		[]string
}

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
	iteration	int
}


var CM ConsensusModule
var iteration int

func NewConsensusModule(id string, peerIds []string) *ConsensusModule {
	CM := new(ConsensusModule)
	CM.id = id
	CM.PeerIds= peerIds
	CM.state = Follower
	CM.votedFor = ""
	CM.nextIndex = make(map[int]int)
	CM.matchIndex = make(map[int]int)
	CM.commitIndex = -1
	CM.lastApplied = -1
	iteration = 1

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

	//dlog(fmt.Sprintf("Submit received by %v: %v", CM.state, command))
	if CM.state == Leader {
		CM.log = append(CM.log, LogEntry{ACLObj: command, Term: CM.currentTerm})
		//dlog(fmt.Sprintf("... log=%v", CM.log))
		CM.currentTerm += 1
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
		if CM.state != Follower && args.LeaderId != CM.id {
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
				CM.log = append(CM.log[:logInsertIndex], args.Entries[newEntriesIndex:]...)
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
	duration := 20
	if len(os.Getenv("RAFT_FORCE_MORE_REELECTION")) > 0 && rand.Intn(3) == 0 {
		return time.Duration(duration) * time.Second
	} else {
		return time.Duration(duration+rand.Intn(150)) * time.Second
	}
}

func runElectionTimer(myid string) {
	timeoutDuration := electionTimeout()
	CM.mu.Lock()
	termStarted := CM.currentTerm
	CM.mu.Unlock()
	dlog(fmt.Sprintf("election timer started (%v), term=%d", timeoutDuration, termStarted))
	CM.id = myid

	ticker := time.NewTicker(5000 * time.Millisecond)
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
			fmt.Println("Haven't heard, starting election")
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
	dlog(fmt.Sprintf("becomes Candidate (currentTerm=%d)", savedCurrentTerm))

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
	dlog(fmt.Sprintf("becomes Follower with term=%d", term))
	CM.state = Follower
	CM.currentTerm = term
	CM.votedFor = ""
	CM.electionResetEvent = time.Now()

	go runElectionTimer(CM.id)
}

func startLeader() {
	CM.state = Leader
	dlog(fmt.Sprintf("becomes Leader; term=%d", CM.currentTerm))

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
		ticker := time.NewTicker(5 * time.Second)
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
	go func() {
		rotateTicker := time.NewTicker(2 * time.Minute)
		defer rotateTicker.Stop()
		for {
			CM.mu.Lock()
			if CM.state != Leader {
				CM.mu.Unlock()
				return
			}
			CM.mu.Unlock()
			if CM.currentTerm > 1 {
				hname, err := os.Hostname()
				if err != nil {
					log.Fatalln("Unable to get hostname")
				}
				postVal := map[string]string{"iteration": strconv.Itoa(iteration), "quorumMems": strconv.Itoa(len(CM.PeerIds) + 1)}
				jsonDat, err := json.Marshal(postVal)
				if err != nil {
					log.Fatalln(err)
				}
				//resp, err := security.TLSPostReq(hname, "/service/rotation/makeCA", "rotation", "application/json", bytes.NewBuffer(jsonDat))
				fmt.Println("Sending makeCA signal to myself")
				resp, err := http.Post("http://" + hname + ":8080/makeCA", "application/json", bytes.NewBuffer(jsonDat))
				if err != nil || resp.StatusCode != http.StatusOK {
					fmt.Println(err)
				}
				fmt.Println(" --- Done with makeCA")
				defer resp.Body.Close()

				// Make gofunc()
				for _, ele := range CM.PeerIds {
					var qMems []string
					for _, mem := range CM.PeerIds {
						if mem != ele {
							qMems = append(qMems, mem)
						}
					}
					qMems = append(qMems, hname)
					postVal := struct {
						Iteration	string
						Prefix		string
						QuorumMems	[]string
					}{Iteration: strconv.Itoa(iteration), Prefix: ele, QuorumMems: qMems}

					jsonDat, err = json.Marshal(postVal)
					if err != nil {
						log.Fatalln(err)
					}
					//CM.PeerIds stores IP addresses, not node names, need to look up so that TLS req can be made properly and verified with cert
					fmt.Println("Sending pullCA signal to quorum member")
					resp, err = security.TLSPostReq(ele, "/service/rotation/pullCA", "rotation", "application/json", bytes.NewBuffer(jsonDat))
					//resp, err = security.TLSPostReq(ele, "/service/rotation/pullCA", "rotation", "application/json", bytes.NewBuffer(jsonDat))
					//resp, err = http.Post("http://" + ele + ":8080/pullCA", "application/json", bytes.NewBuffer(jsonDat))
					if err != nil || resp.StatusCode != http.StatusOK {
						fmt.Println(err)
						fmt.Printf("Failure to notify other Quorum members of CA artifacts\n")
					}
					defer resp.Body.Close()
					fmt.Println(" --- Done with pullCA")
				}
				var newMap AssignmentMap
				fullMap := processManifest(newMap)
				splitMap := splitAssignments(len(CM.PeerIds)+1, fullMap)
				//splitAssignments(len(CM.PeerIds)+3, fullMap)

				// Make gofunc()
				var semaphore = make(chan struct{}, len(CM.PeerIds)+1)
				for i:=0; i < len(CM.PeerIds)+1; i++ {
					semaphore <- struct{}{}
					if i == len(CM.PeerIds) {
						splitMap[i].Iteration = iteration
						splitMap[i].Prefix = hname
						splitMap[i].Gossip = true
						prepVal := splitMap[i]
						jsonDat, err = json.Marshal(prepVal)
						if err != nil {
							log.Fatalln(err)
						}
						//resp, err = security.TLSPostReq(hname, "/service/rotation/assignment", "rotation", "application/json", bytes.NewBuffer(jsonDat))
						fmt.Println("Sending assignment to myself")
						resp, err = http.Post("http://" + hname + ":8080/assignment", "application/json", bytes.NewBuffer(jsonDat))
						if err != nil || resp.StatusCode != http.StatusOK {
							fmt.Println(err)
							fmt.Printf("Failure to send generation assignment to self\n")
						}
						defer resp.Body.Close()
						fmt.Println(" --- Done sending assignment to self")
						<-semaphore
					} else {
						splitMap[i].Iteration = iteration
						splitMap[i].Prefix = CM.PeerIds[i]
						splitMap[i].Gossip = false
						prepVal := splitMap[i]
						jsonDat, err = json.Marshal(prepVal)
						if err != nil {
							log.Fatalln(err)
						}
						fmt.Println("Sending assignment to quorum member")
						resp, err = security.TLSPostReq(CM.PeerIds[i], "/service/rotation/assignment", "rotation", "application/json", bytes.NewBuffer(jsonDat))
						//resp, err = http.Post("http://" + CM.PeerIds[i] + ":8080/assignment", "application/json", bytes.NewBuffer(jsonDat))
						if err != nil || resp.StatusCode != http.StatusOK {
							fmt.Println(err)
							fmt.Printf("Failure to send generation assignments to other quorum members\n")
						}
						defer resp.Body.Close()
						fmt.Println(" --- Done sending assignment to quorum member")
						<-semaphore
					}
				}

				for i:=0; i < len(CM.PeerIds)+1; i++ {
					semaphore <- struct{}{}
					if i == len(CM.PeerIds) {
						resp, err := security.TLSGetReq(hname, "/anvil/raft/peerList", "")
						b, err := ioutil.ReadAll(resp.Body)
						if err != nil {
							fmt.Println(err)
						}
						defer resp.Body.Close()
						var temptargets []string
						err = json.Unmarshal(b, &temptargets)
						if err != nil {
							log.Println(err)
						}
						targets := []string{}
						for _,e := range temptargets {
							if e != hname {
								addr, err := net.LookupIP(e)
								if err != nil {
									fmt.Println("Lookup failed")
								}
								targets = append(targets, addr[0].String())
							}
						}
						collectMap := struct {
							Targets         []string
							Iteration       string
							QuorumMems	[]string
						}{Targets: targets, Iteration: strconv.Itoa(iteration), QuorumMems: CM.PeerIds}

						jsonData, err := json.Marshal(collectMap)
						if err != nil {
							log.Fatalln("Unable to marshal JSON")
						}
						fmt.Println("Sending collection signal to myself")
						resp, err = http.Post("http://" + hname + ":8080/collectSignal", "application/json", bytes.NewBuffer(jsonData))
						defer resp.Body.Close()
						_, err = ioutil.ReadAll(resp.Body)
						if err != nil {
							fmt.Println("Bad Read")
						}
						fmt.Println(" --- Done sending collection signal to myself")
						<-semaphore
					} else {
						var qMems []string
						sendTarg := CM.PeerIds[i]
						for _, ele := range CM.PeerIds {
							if ele != sendTarg {
								qMems = append(qMems, ele)
							}
						}
						qMems = append(qMems, hname)
						var targets []string
						copy(targets, CM.PeerIds[:0])
						hostAddr, err := net.LookupIP(hname)
						if err != nil {
							fmt.Println("Lookup failed")
						}
						for _,t := range CM.PeerIds {
							targAddr, err := net.LookupIP(sendTarg)
							if err != nil {
								fmt.Println("Lookup failed")
							}
							if t != sendTarg && t != targAddr[0].String() && t != hname && t != hostAddr[1].String() {
								addr, err := net.LookupIP(t)
								if err != nil {
									fmt.Println("Lookup failed")
								}
								targets = append(targets, addr[0].String())
							}
						}
						targets = append(targets, hostAddr[1].String())

						collectMap := struct {
							Targets         []string
							Iteration       string
							QuorumMems	[]string
						}{Targets: targets, Iteration: strconv.Itoa(iteration), QuorumMems: qMems}

						jsonData, err := json.Marshal(collectMap)
						if err != nil {
							log.Fatalln("Unable to marshal JSON")
						}
						fmt.Println("Sending collection signal to quorum member")
						resp, err = security.TLSPostReq(sendTarg, "/service/rotation/collectSignal", "rotation", "application/json", bytes.NewBuffer(jsonData))
						defer resp.Body.Close()
						_, err = ioutil.ReadAll(resp.Body)
						if err != nil {
							fmt.Println("Bad Read")
						}
						fmt.Println(" --- Done sending collection signal to quorum member")
						<-semaphore
					}
				}

				aclEntries,_ := acl.ACLIngest("/root/anvil-rotation/artifacts/"+strconv.Itoa(iteration)+"/acls.yaml")
				for _, ele := range aclEntries {
					postBody, _ := json.Marshal(ele)
					responseBody := bytes.NewBuffer(postBody)
					//resp, err := http.Post("http://"+hname+":443/anvil/raft/pushACL", "application/json", responseBody)
					fmt.Println("Ingesting the acls file into raft")
					resp, err := security.TLSPostReq(hname, "/anvil/raft/pushACL", "", "application/json", responseBody)
					defer resp.Body.Close()
					if err != nil {
						log.Fatalln("Unable to post content")
					}
					_, err = ioutil.ReadAll(resp.Body)
					if err != nil {
						log.Fatalln("Unable to read received content")
					}
					fmt.Println(" --- Done ingesting the acls file into raft")
				}

				resp, err = security.TLSGetReq(hname, "/anvil/catalog/clients", "")
				if err != nil {
					fmt.Println(err)
				}
				defer resp.Body.Close()

				b, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Println(err)
				}
				clientList := struct {
					Clients		[]string
				}{}
				err = json.Unmarshal(b, &clientList)
				if err != nil {
					fmt.Println(err)
				}


				semaphore = make(chan struct{}, len(clientList.Clients))
				for _, ele := range clientList.Clients {
					semaphore <- struct{}{}
                                        //postVal = map[string]string{"iteration": strconv.Itoa(iteration), "prefix": ele}
					postVal := struct {
						Iteration	string
						Prefix		string
						QuorumMems	[]string
					}{Iteration: strconv.Itoa(iteration), Prefix: ele, QuorumMems: append(CM.PeerIds, hname)}
                                        jsonDat, err = json.Marshal(postVal)
                                        if err != nil {
                                                log.Fatalln(err)
                                        }
					fmt.Println("Notifying client to pull rotation artifacts")
                                        resp, err = security.TLSPostReq(ele, "/anvil/rotation", "", "application/json", bytes.NewBuffer(jsonDat))
					defer resp.Body.Close()
                                        if err != nil || resp.StatusCode != http.StatusOK {
                                                fmt.Printf("Failure to notify all clients of available artifacts\n")
                                        }
					_, err = ioutil.ReadAll(resp.Body)
					if err != nil {
						fmt.Println("Bad Read")
					}
					fmt.Println(" --- Done notifying client to pull rotation artifacts")
					<-semaphore
                                }

				semaphore = make(chan struct{}, len(CM.PeerIds)+1)
				for i:=0; i < len(CM.PeerIds)+1; i++ {
					semaphore <- struct{}{}
					if i == len(CM.PeerIds) {
						fmt.Println("Telling myself to change my config")
						_, err = security.TLSGetReq(hname, "/anvil/rotation/config", "")
						if err != nil {
							fmt.Println(err)
							log.Println("Failed to adjust leader config")
						}
						fmt.Println(" --- Done telling myself to change my config")
						<-semaphore
					} else {
						sendTarg := CM.PeerIds[i]
						fmt.Println("Telling quorum member to change config")
						_, err = security.TLSGetReq(sendTarg, "/anvil/rotation/config", "")
						if err != nil {
							fmt.Println(err)
							log.Println("Failed to adjust quorum configs")
						}
						fmt.Println(" --- Done telling quorum member to change config")
						<-semaphore
					}
				}

				semaphore = make(chan struct{}, len(clientList.Clients))
                                for _, ele := range clientList.Clients {
                                        semaphore <- struct{}{}
					fmt.Println("Telling client to adjust config")
                                        resp, err = security.TLSGetReq(ele, "/anvil/rotation/config", "")
                                        defer resp.Body.Close()
                                        if err != nil || resp.StatusCode != http.StatusOK {
						fmt.Println(err)
                                                fmt.Printf("Failure to notify all clients of available artifacts\n")
                                        }
					fmt.Println(" --- Done telling client to change config")
                                        <-semaphore
                                }

				iteration = iteration + 1
			}
			<-rotateTicker.C
			fmt.Println("Rotating . . .")
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
        resp, err := security.TLSGetReq(target, "/anvil/catalog/leader", "")
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

	//resp, err := http.Post("http://" + target + ":443/anvil/raft/updateleader", "application/json", postBody)
	resp, err := security.TLSPostReq(target, "/anvil/raft/updateleader", "", "application/json", postBody)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	return
}

func SendAppendEntry(target string, args AppendEntriesArgs) (error, AppendEntriesReply) {
        reqBody, _ := json.Marshal(args)
        postBody := bytes.NewBuffer(reqBody)
	//resp, err := http.Post("http://" + target + ":443/anvil/raft/appendentries", "application/json", postBody)
	resp, err := security.TLSPostReq(target, "/anvil/raft/appendentries", "", "application/json", postBody)
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
	//resp, err := http.Get("http://" + leader + ":443/anvil/raft/backlog/" + strconv.Itoa(CM.commitIndex))
	resp, err := security.TLSGetReq(leader, "/anvil/raft/backlog/" + strconv.Itoa(CM.commitIndex), "")
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
	//resp, err := http.Post("http://" + target + ":443/anvil/raft/requestvote", "application/json", postBody)
	resp, err := security.TLSPostReq(target, "/anvil/raft/requestvote", "", "application/json", postBody)
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

func processManifest(aMap AssignmentMap) (AssignmentMap) {
	yamlFile, err := ioutil.ReadFile("./config/manifest.yaml")
        if err != nil {
                log.Printf("Read file error #%v", err)
        }
        err = yaml.Unmarshal(yamlFile, &aMap)
        if err != nil {
                log.Fatalf("Unmarshal: %v", err)
        }
	return aMap
}

func splitAssignments(numPeers int, aMap AssignmentMap) ([]AssignmentMap) {
	lastNode := 0.0
	lastSvc := 0.0
	splitMap := make([]AssignmentMap, numPeers)
	nodeQuantity := float64(len(aMap.Nodes)) / float64(numPeers)
	svcQuantity := float64(len(aMap.SvcMap)) / float64(numPeers)
	for i:=0; i < numPeers; i++ {
		var tMap AssignmentMap
		tMap.Quorum = aMap.Quorum
		tMap.Nodes = aMap.Nodes[int(lastNode):int(lastNode+nodeQuantity)]
		tMap.SvcMap = aMap.SvcMap[int(lastSvc):int(lastSvc+svcQuantity)]
		lastNode = lastNode+nodeQuantity
		lastSvc = lastSvc+svcQuantity
		splitMap[i] = tMap
	}
	return splitMap
}
