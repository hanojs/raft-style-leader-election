package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"
	"time"
)

//The period for heartbeats in ms
const THeatbeat = 5000

//Enums for the state
const (
	Leader    = 0
	Follower  = 1
	Candidate = 2
)

type Task int

type Vote struct {
	Id   int
	Vote bool
}

type Node struct {
	id       int
	state    int
	port     int
	leader   int
	voted    int
	term     int
	lastPing int64
	timeout  int
}

type Heartbeat struct {
	Id     int
	Term   int
	Leader int
}

var clientRpcList []*rpc.Client
var clientVotes []bool

var thisNode Node
var s1 = rand.NewSource(time.Now().UnixNano())
var r1 = rand.New(s1)
var killLeader bool

func main() {

	//Parsing command line arguments
	serverPortInitial := flag.Int("portBlockStart", 8000, "The prt address to start incrementing from")
	nServers := flag.Int("numServers", 0, "Number of servers in the network, used in conjunction with portBlockStart to determine what poorts to connect to")
	port := flag.Int("port", 0, "Port for this node to occupy")
	killLeader1 := flag.Bool("killLeader", false, "When specified, a leader will crash 8 seconds after sending it's first heartbeat")
	flag.Parse()
	killLeader = *killLeader1
	
	//Registering the server for rpc
	task := new(Task)
	rpc.Register(task)
	rpc.HandleHTTP()

	//Setting up the lists to a dynamic size
	clientRpcList = make([]*rpc.Client, *nServers)
	clientVotes = make([]bool, *nServers)
	
	//Setting up the node
	thisNode.state = Follower
	thisNode.port = *port
	thisNode.id = thisNode.port - *serverPortInitial
	thisNode.term = 0
	thisNode.voted = -1

	log.Println("Initialized Id    " + strconv.Itoa(thisNode.id))
	log.Println("Initialized State " + strconv.Itoa(thisNode.state))

	//Start listening for rpcs
	listener, _ := net.Listen("tcp", ":"+strconv.Itoa(thisNode.port))
	defer listener.Close()

	go http.Serve(listener, nil)

	//Connect to all other servers
	for i := 0; i < *nServers; i++ {
		if i+*serverPortInitial == thisNode.port { //Don't connect to yourself...
			continue
		}
		//Tries to connect to the specified port once, if there's any error
		// it logs the error and keeps trying until it connects.
		// If there was no error, it continues on.
		client, err := rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(i+*serverPortInitial))
		if err != nil {
			log.Println("Connection error:", err.Error())
			for err != nil {
				client, err = rpc.DialHTTP("tcp", "localhost:"+strconv.Itoa(i+*serverPortInitial))
			}
		}
		defer client.Close() //Closes the connection if the client is dead
		clientRpcList[i] = client
		clientVotes[i] = false
	}
	thisNode.lastPing = timestamp(time.Now())
	thisNode.timeout = getRandTimeout()
	for {
		switch thisNode.state {
		case Leader:
			runLeader()
		case Follower:
			checkFollower()
		case Candidate:
			runCandidate()
		}
	}

}

//RPC - Server - processing a received heartbeat
func (t *Task) ReceiveHeartbeat(args *Heartbeat, reply *Heartbeat) error {
	if args.Term > thisNode.term {
		thisNode.lastPing = timestamp(time.Now())
		thisNode.leader = args.Leader
		thisNode.state = Follower
		thisNode.term = args.Term
		thisNode.timeout = getRandTimeout()
	} else if thisNode.state == Follower && args.Id == thisNode.leader {
		thisNode.lastPing = timestamp(time.Now())
		thisNode.timeout = getRandTimeout()
	}
	reply.Term = thisNode.term
	reply.Leader = thisNode.leader
	reply.Id = thisNode.id
	log.Println("Process: " + strconv.Itoa(thisNode.id) + " Heartbeat recieved from process: " + strconv.Itoa(args.Id))
	return nil
}

//RPC - Server - processing a received vote request
func (t *Task) RecieveVoteRequest(sender *Heartbeat, reply *Vote) error {
	reply.Id = thisNode.id
	if thisNode.term < sender.Term {
		thisNode.voted = sender.Id
		reply.Vote = true
	} else {
		reply.Vote = false
	}
	log.Println("Process: " + strconv.Itoa(reply.Id) + " Vote request recieved from process: " + strconv.Itoa(sender.Id) + " Voting: " + strconv.FormatBool(reply.Vote))
	return nil
}

// Handles the server when it is in candidate mode
func runCandidate() {
	debugVar := timestamp(time.Now()) - thisNode.lastPing
	if debugVar > int64(thisNode.timeout) {
		log.Println("Node: " + strconv.Itoa(thisNode.id) + " :: Timeout achieved as a candidate, term increasing and requesting votes, term: " + strconv.Itoa(thisNode.term))
		moveToCandidate()
		return
	}
	neededVotes := ceilDivideByTwo(len(clientVotes)) + 1 //Calculation to see if this node has enough votes to be elected leader majority of nodes 
	for _, vote := range clientVotes {
		if vote {
			neededVotes--
		}
	}
	if neededVotes <= 0 {
		thisNode.state = Leader
		log.Println("Node: " + strconv.Itoa(thisNode.id) + " :: Received majority of votes, becoming leader")
	}
}

// Handler for when the node is in a follower state
func checkFollower() {
	if timestamp(time.Now()) - thisNode.lastPing > int64(thisNode.timeout) { //If the node has not received a heartbeat in the randomized timeout period, move to the candidate stage
		thisNode.state = Candidate
		log.Println("Node: " + strconv.Itoa(thisNode.id) + " :: Timeout achieved, beoming candidate and requesting votes, term: " + strconv.Itoa(thisNode.term))
		
		moveToCandidate() //Moving to the candidate stage, and voting for itself
	}
}

// Transition function to move the node to the candidate stage. 
func moveToCandidate() {
	for i, _ := range clientVotes { //Clearing out any old votes
		clientVotes[i] = false
	}
	thisNode.term++
	thisNode.leader = thisNode.id
	thisNode.voted = thisNode.id
	clientVotes[thisNode.id] = true
	requestVoteMulticast() //requesting votes from everyone
	thisNode.lastPing = timestamp(time.Now())
	thisNode.timeout = getRandTimeout()
}

// Handler for when the node is in a leader state
func runLeader() {
	ticker := time.NewTicker(THeatbeat / 1000 * time.Second)
	quit := make(chan struct{})
	func() {
		for {
			select {
			case <-ticker.C:
				sendHeartbeat()
				go if killLeader { // If the killleader option is enabled, a leader will crash 8 seconds after it sends its first heartbeat.
					os.Exit(8)
				}
				return
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
}

// Turns a time into milliseconds
func timestamp(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// Sends a heartbeat to all clients
func sendHeartbeat() {
	for _, client := range clientRpcList {
		if client == nil {
			continue
		}
		go func(client *rpc.Client) {
			arg := Heartbeat{Term: thisNode.term, Leader: thisNode.leader, Id: thisNode.id}
			var response Heartbeat
			client.Call("Task.ReceiveHeartbeat", arg, &response)
			log.Println("Response heartbeat - term: " + strconv.Itoa(response.Term) + " Leader: " + strconv.Itoa(response.Leader) + " Id: " + strconv.Itoa(response.Id))
			if response.Term > thisNode.term {
				thisNode.state = Follower
				thisNode.leader = response.Leader
				thisNode.term = response.Term
				thisNode.lastPing = timestamp(time.Now())
			}
		}(client)
	}
}

// Gets a timeout val somehwhere from heartbeat to 5/3 heartbeat time (This range was designed to sometimes, if rarely, cause a false positive leader failure)
func getRandTimeout() int {
	return r1.Intn(2*THeatbeat/3) + THeatbeat
}

// Sends a multicast to all nodes
func requestVoteMulticast() {
	for _, client := range clientRpcList {
		if client == nil {
			continue
		}
		go func(client *rpc.Client) { //Kicks off a vote request that will run concurrently
			arg := Heartbeat{Term: thisNode.term, Leader: thisNode.leader, Id: thisNode.id}
			var response Vote
			client.Call("Task.RecieveVoteRequest", arg, &response)
			clientVotes[response.Id] = response.Vote
			log.Println("Vote response :: ID: " + strconv.Itoa(response.Id) + ", Vote: " + strconv.FormatBool(response.Vote))
		}(client) //Passing in the client variable. Needed for each coroutine to take the corect client variable at time of its execution.
	}
}

//Returns an int divided by two, rounded up
func ceilDivideByTwo(num int) int {
	val := float64(num) / 2.0
	if num%2 > 0 {
		val++
	}
	return int(val)
}
