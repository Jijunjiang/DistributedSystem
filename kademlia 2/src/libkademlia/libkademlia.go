package libkademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"container/list"
	"os"
	"time"

	"errors"
)

const (
	alpha = 3
	b     = 8 * IDBytes
	knum  = 20

)



// Kademlia type. You can put whatever state you need in this.
type Kademlia struct {
	NodeID      	ID
	SelfContact 	Contact
	DataMap 		map[ID][]byte				// Data map stored in the Node
	Table 			*RoutingTable 				// Routing table
	Countsum 		int					 //count if no node to do ietration
	findContactChan     chan ID
	findResultChan	    chan Contact
	ContactUpdateChan   chan *Contact 			// Contact channel
	DataStoreChan 		chan *Data				// Data channel
	DataResChan		chan string				//signifying the operation
	BucketIndexChan 	chan int 				// Bucket Index (ID) channel
	BucketChan 			chan *list.List 		// Bucket list channel
	NodeSearchInChan 	chan ID
	NodeSearchOutChan  	 chan []Contact
	ValueSearchInChan	chan ID

	ValueConfirmChan    chan int
	ValueSearchDoChan   chan ID
	ValueResultChan     chan []byte
	ValueSearchNodeChan chan ID
	ValueNodeResult     chan []Contact
	////////////////////////////////////////////////project2/////////////////////////////

	ShortList	SList
	iterativeFindResChan   chan sContact
	iterativeFindValueChan  chan vContact
	iterativeFindValueResChan chan []byte

	countNoNodeIterationChan	chan int
	TakeCountNoNodeInChan		chan bool
	TakeCountNoNodeOutChan		chan int
	iterV				[]byte


	TakeItChanIn	chan bool
	TakeItChanOut	chan []byte
	//////////////////////////////////////////////////////////////////////////////////////
}

// Data tuple
type Data struct {
	Key ID
	Value []byte
}

func NewKademliaWithId(laddr string, nodeID ID) *Kademlia {
	k := new(Kademlia)
	k.NodeID = nodeID

	k.Countsum = 0

	// TODO: Initialize other state here as you add functionality.

	// Set up RPC server
	// NOTE: KademliaRPC is just a wrapper around Kademlia. This type includes
	// the RPC functions.

	s := rpc.NewServer()
	s.Register(&KademliaRPC{k})
	hostname, port, err := net.SplitHostPort(laddr)
	if err != nil {
		return nil
	}
	s.HandleHTTP(rpc.DefaultRPCPath+port,
		rpc.DefaultDebugPath+port)
	l, err := net.Listen("tcp", laddr)
	if err != nil {
		log.Fatal("Listen: ", err)
	}

	// Run RPC server forever.
	go http.Serve(l, nil)

	// Add self contact
	hostname, port, _ = net.SplitHostPort(l.Addr().String())
	port_int, _ := strconv.Atoi(port)
	ipAddrStrings, err := net.LookupHost(hostname)
	var host net.IP
	for i := 0; i < len(ipAddrStrings); i++ {
		host = net.ParseIP(ipAddrStrings[i])
		if host.To4() != nil {
			break
		}
	}

	k.SelfContact = Contact{k.NodeID, host, uint16(port_int)}
	// Initialize routing table
	k.Table = NewRoutingTable(k.SelfContact)
	// Initialize data map
	k.DataMap = make(map[ID][]byte)

	// Initialize channels
	k.findContactChan = make(chan ID)
	k.findResultChan = make(chan Contact)

	k.ContactUpdateChan = make(chan *Contact)

	k.DataStoreChan = make(chan *Data)
	k.DataResChan = make(chan string)

	k.BucketIndexChan = make(chan int)
	k.BucketChan = make(chan *list.List)

	k.NodeSearchInChan = make(chan ID)
	k.NodeSearchOutChan  = make(chan []Contact)

	k.ValueSearchInChan = make(chan ID)

	k.ValueConfirmChan  = make(chan int)
	k.ValueSearchDoChan  = make (chan ID)
	k.ValueResultChan  =   make(chan []byte)
	k.ValueSearchNodeChan = make(chan ID)
	k.ValueNodeResult   =  make(chan []Contact)
	///////////////////////////////////////////////project2/////////////////////////////

	k.iterativeFindResChan = make(chan sContact)
	k.iterativeFindValueChan = make(chan vContact)
	k.iterativeFindValueResChan = make(chan []byte)
	k.countNoNodeIterationChan = make(chan int)
	k.TakeCountNoNodeInChan = make(chan bool)
	k.TakeCountNoNodeOutChan = make(chan int)
	k.iterV = make([]byte, 0)

	k.TakeItChanIn = make(chan bool)
	k.TakeItChanOut = make(chan []byte)
	////////////////////////////////////////////////////////////////////////////////////
	go k.ChannelHandler()

	return k
}

func NewKademlia(laddr string) *Kademlia {
	return NewKademliaWithId(laddr, NewRandomID())
}

type ContactNotFoundError struct {
	id  ID
	msg string
}

func (e *ContactNotFoundError) Error() string {
	return fmt.Sprintf("%x %s", e.id, e.msg)
}

// Handle channel for data updating
func (k *Kademlia) ChannelHandler() {
	num := 20
	for {
		select {
		// Update routing table after successful communication

		case targetContact := <-k.findContactChan:
			target := k.Table.FindContact(targetContact)
			k.findResultChan <- target

		case contact := <-k.ContactUpdateChan:
			k.Table.Update(contact)
			// Access bucket using prefix_length as index
		case prefix_length := <-k.BucketIndexChan:
			k.BucketChan <- k.Table.KBuckets[prefix_length]
			// Store new Key-Value pair
		case data := <-k.DataStoreChan:
			_, ok := k.DataMap[data.Key] //if the key is already existed
			if ok {
				k.DataResChan <- "Key already existed"
			} else {
				k.DataMap[data.Key] = data.Value
				k.DataResChan <- "Stored success!"
			}

			// channel stores node is, and find the k nearest nodes and return to out channal
		case nodeTofind := <-k.NodeSearchInChan:
			//ret := make([]Contact, 0, num)
			ret := k.Table.FindClosest(nodeTofind, num)
			k.NodeSearchOutChan <- ret
			//
		case keyToSearch := <-k.ValueSearchInChan:
			_, ok := k.DataMap[keyToSearch]
			if ok {
				valueRes := k.DataMap[keyToSearch]
				k.ValueResultChan <- valueRes
			} else {
				k.ValueResultChan <- nil
			}

		case valueNode := <-k.ValueSearchNodeChan:
			ret := make([]Contact, 0, num)
			ret = k.Table.FindClosest(valueNode, num)
			k.ValueNodeResult <- ret

		case contactFound := <-k.iterativeFindResChan:
			if contactFound.dead == false{
				k.ShortList.ExpandShortListChan <- contactFound.returnContact
				sender := new(Contact)
				sender.Port = contactFound.senderPort
				sender.Host = contactFound.senderHost
				sender.NodeID = contactFound.senderNodeID
				k.ShortList.MoveToActiveChan <- *sender
			}

		case contactFound := <-k.iterativeFindValueChan:
			if contactFound.dead == false {
				if contactFound.returnValue != nil {
					fmt.Println("Really found!")
					fmt.Println(contactFound.returnValue)
					k.DoStore(&k.ShortList.ClosestNode, contactFound.SKey, contactFound.returnValue)

					for _,num :=range(contactFound.returnValue) {
						k.iterV = append(k.iterV,num)
					}
					fmt.Println("k.iterVPRE",k.iterV)
				}


					k.ShortList.ExpandShortListChan <- contactFound.returnContact
					sender := new(Contact)
					sender.Port = contactFound.senderPort
					sender.Host = contactFound.senderHost
					sender.NodeID = contactFound.senderNodeID
					k.ShortList.MoveToActiveChan <- *sender
					//k.iterativeFindValueResChan <- nil

			}
		case count := <-k.countNoNodeIterationChan:
			k.Countsum = k.Countsum + count

		case TakeCOunt := <-k.TakeCountNoNodeInChan:
			//fmt.Println("Coll")
			if (TakeCOunt) {
				k.TakeCountNoNodeOutChan <- k.Countsum
				//fmt.Println("Yoo")

			}

		case TakeIout :=<- k.TakeItChanIn:
			if TakeIout {
				k.TakeItChanOut <- k.iterV
			}
		}
	}
}

// Get bucket list according to prefix_length
func (k *Kademlia) getBucket(prefix_length int) (ret *list.List) {
	k.BucketIndexChan <- prefix_length
	ret = <- k.BucketChan
	return
}

func (k *Kademlia) FindContact(nodeId ID) (*Contact, error) {
	// TODO: Search through contacts, find specified ID
	// Find contact with provided ID
	/*
	input: nodeid
	output: contact, error
	functionality: Find contact with provided ID
	channel in: findContactChan
	channel out: findResultChan
	*/
	k.findContactChan <- nodeId
	findRes := <-k.findResultChan

	if nodeId == k.SelfContact.NodeID {
		return &k.SelfContact, nil
	}else if findRes.Port !=0 {
		return &findRes, nil
	}

	return nil, &ContactNotFoundError{nodeId, "Not found"}
}

type CommandFailed struct {
	msg string
}

func (e *CommandFailed) Error() string {
	return fmt.Sprintf("%s", e.msg)
}

func (k *Kademlia) DoPing(host net.IP, port uint16) (*Contact, error) {

	/* do ping function
		input: target IP address, port number
		output: error message and taget contact
		functionality:build RPC connection, send ping message(sender, messageID), receive pong message(sender, messageID)
		update bucket of sender
		channel in: ContactUpdateChan
	*/

	//build RPC connection
	hostname := host.String()
	portnum := strconv.Itoa(int(port))
	address := hostname + ":" + portnum

	//generate ping, call server and receive pong
	ping := &PingMessage{
		Sender : k.SelfContact,
		MsgID : NewRandomID(),
	}
	var pong PongMessage



	client, err := rpc.DialHTTPPath("tcp", address, rpc.DefaultRPCPath+portnum)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , DoPingDial",err.Error())
	}
	//call other node's rpc function ping

	err = client.Call("KademliaRPC.Ping", ping, &pong)


	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , DoPingCall",err.Error())
	}

	// no error update bucket and return
	k.ContactUpdateChan <- &pong.Sender

	return &pong.Sender,err

	// return nil, &CommandFailed{
	// 	"Unable to ping " + fmt.Sprintf("%s:%v", host.String(), port)}
}

func (k *Kademlia) DoStore(contact *Contact, key ID, value []byte) error {

	/* Do store function
	   input: taget contact
	   		  request key and value
	   output: error
	   functionality: build RCP connection, send store request, receive response
	*/

	// build RPC link to target host
	hostname := contact.Host.String()
	portnum := strconv.Itoa(int(contact.Port))
	address := hostname + ":" + portnum

	client, err := rpc.DialHTTPPath("tcp", address, rpc.DefaultRPCPath+portnum)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , DoStoreDial",err.Error())
	}

	//generate store request, call server and recive response
	storeReq := &StoreRequest{
		Sender : k.SelfContact,
		MsgID  : NewRandomID(),
		Key    : key,
		Value  : value,
	}

	var storeRes StoreResult
	//call other node's rpc function ping
	err = client.Call("KademliaRPC.Store", storeReq, &storeRes)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , DoStoreCall",err.Error())
	}

	//return info showing the status of storing
	fmt.Println(storeRes.OperationInfo)
	return nil

	//return &CommandFailed{"Not implemented"}
}

func (k *Kademlia) DoFindNode(contact *Contact, searchKey ID) ([]Contact, error) {

	/*do find node function
	 input: RPC target contact, Serarch ID
	 output: list of contact
	 functionality:
	 linked to RPC server with targte contact, call find node function in RPC server
	 receive response(list of node) and update the bucket
	 channel in: ContactUpdateChan
	*/


	hostname := contact.Host.String()
	portnum := strconv.Itoa(int(contact.Port))
	address := hostname + ":" + portnum

	client, err := rpc.DialHTTPPath("tcp", address, rpc.DefaultRPCPath+portnum)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , DoFindNodeDial",err.Error())
	}

	//generate find node request, call server and recive response(up to 20 nodes)
	req := &FindNodeRequest{
		Sender : k.SelfContact,
		MsgID : NewRandomID(),
		NodeID : searchKey,
	}
	//call other node's rpc function ping

	var res FindNodeResult
	err = client.Call("KademliaRPC.FindNode", req, &res)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , DoFindNodeCall",err.Error())
	}

	if res.Nodes != nil{
		for index := 0; index < len(res.Nodes); index++ {
			k.ContactUpdateChan <- &res.Nodes[index]
		}
	}

	return res.Nodes, err

	//return nil, &CommandFailed{"Not implemented"}
}

func (k *Kademlia) DoFindValue(contact *Contact,
	searchKey ID) (value []byte, contacts []Contact, err error) {
	// TODO: Implement

	/* fo find value function
	 input: RPC target contact, Serarchkey
	 output: value, list of contact
	 functionality:
	 linked to RPC server with targte contact, call find value function in RPC server
	 receive response(value, list of node) and update the bucket
	 Channel in: ContactUpdateChan
	*/


	hostname := contact.Host.String()
	portnum := strconv.Itoa(int(contact.Port))
	address := hostname + ":" + portnum

	client, err := rpc.DialHTTPPath("tcp", address, rpc.DefaultRPCPath+portnum)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , DoFindValueDial",err.Error())
	}


	//generate find value request and call the server FindValue function, receive respons(FindValueResult)
	FindValueReq := FindValueRequest{
		Sender : *contact,
		MsgID  : NewRandomID(),
		Key    : searchKey,
	}

	var FindValueRes FindValueResult

	err = client.Call("KademliaRPC.FindValue", FindValueReq, &FindValueRes)

	//update bucket if the response is not null
	if FindValueRes.Nodes != nil {
		for index := 0; index < len(FindValueRes.Nodes); index++ {
			k.ContactUpdateChan <- &FindValueRes.Nodes[index]
		}
	}else{
		fmt.Sprintf(string(FindValueRes.Value))
	}

	return FindValueRes.Value, FindValueRes.Nodes, nil

	//return nil, nil, &CommandFailed{"Not implemented"}
}

func (k *Kademlia) LocalFindValue(searchKey ID) ([]byte, error) {

	/*
	find value in self data map
	input: key
	output: value, error(nil if not)
	*/

	_, ok := k.DataMap[searchKey]
	var valueLocal []byte
	if ok {
		valueLocal = k.DataMap[searchKey]
	}else{
		valueLocal = nil
		fmt.Sprintf("No key found")
	}

	return valueLocal, nil
}
////////////////////////////////////////////////////////project2//////////////////////
func (k * Kademlia) LocalFindNode(nodeId ID) (error) {
	//local to find alpha nodes
	ret := k.Table.FindClosest(nodeId, alpha)
	k.ShortList.ClosestNode = ret[0]
	k.ShortList.ExpandShortListChan <- ret
	return nil
}
// For project 2!

// for iterative call node
type sContact struct{
	returnContact  []Contact
	senderNodeID    ID
	senderHost      net.IP
	senderPort      uint16
	dead	bool
}




func (k *Kademlia) IterativeCallNode(contact Contact, searchID ID) {
	hostname := contact.Host.String()
	portnum := strconv.Itoa(int(contact.Port))
	address := hostname + ":" + portnum

	client, err := rpc.DialHTTPPath("tcp", address, rpc.DefaultRPCPath+portnum)
	if err != nil {
		//fmt.Fprintf(os.Stderr,"Fatal error: %s , DoFindNodeDial",err.Error())
		resContact := new(sContact)
		resContact.returnContact = nil
		resContact.senderNodeID = contact.NodeID
		resContact.senderHost = contact.Host
		resContact.senderPort = contact.Port
		resContact.dead = true
		println("dead",contact.NodeID.AsString())
		k.iterativeFindResChan <- *resContact
		return
	}

	//generate find node request, call server and recive response(up to 20 nodes)
	req := &FindNodeRequest{
		Sender : k.SelfContact,
		MsgID : NewRandomID(),
		NodeID : searchID,
	}
	//call other node's rpc function ping


	var res FindNodeResult
	err = client.Call("KademliaRPC.FindNode", req, &res)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , DoFindNodeCall",err.Error())
	}
	// change the searchID state in visited list to be true


	if res.Nodes != nil{
		//update ping
		for index := 0; index < len(res.Nodes); index++ {
			k.ContactUpdateChan <- &res.Nodes[index]
		}

		resContact := new(sContact)
		resContact.returnContact = res.Nodes
		resContact.senderNodeID = contact.NodeID
		resContact.senderHost = contact.Host
		resContact.senderPort = contact.Port
		resContact.dead = false

		k.iterativeFindResChan <- *resContact

	}else {
		resContact := new(sContact)
		resContact.returnContact = nil
		resContact.senderNodeID = contact.NodeID
		resContact.senderHost = contact.Host
		resContact.senderPort = contact.Port
		resContact.dead = false

		k.iterativeFindResChan <- *resContact
	}

	return
}

func (k *Kademlia) ParallelAlphaNode (){

	NodeToCall := make([]Contact, 0)
	k.ShortList.TakeNonActiveListChanIn <- true
	nonactivelist := <- k.ShortList.TakeNonActiveListChanOut
	length := len(nonactivelist)
	if length >= alpha {
		NodeToCall = nonactivelist[:alpha]
		//delete
		k.ShortList.DeleteChan <- alpha

	} else if length > 0{

		NodeToCall = nonactivelist[:]

		k.ShortList.DeleteChan <- len(nonactivelist)
	} else {
		println("no node to do iteratoin, trying count:", k.Countsum)
		k.countNoNodeIterationChan <- 1
		return

	}

	//call alpha node to find the target id

	for index, _ := range (NodeToCall) {
		newNode := NodeToCall[index]
		go k.IterativeCallNode(newNode, k.ShortList.Target)
	}

}



func (k *Kademlia) ParallelAlphaNodeForValue (){
	// parallelism call alpha nodes after 300 milli
	NodeToCall := make([]Contact, 0)
	k.ShortList.TakeNonActiveListChanIn <- true
	nonactivelist := <- k.ShortList.TakeNonActiveListChanOut
	length := len(nonactivelist)
	if length >= alpha {
		NodeToCall = nonactivelist[:alpha]
		//delete
		k.ShortList.DeleteChan <- alpha
		fmt.Println("delete",1)

	} else if length > 0{

		NodeToCall = nonactivelist[:]

		k.ShortList.DeleteChan <- len(nonactivelist)
		fmt.Println("delete",2)
	} else {
		println("no node to do iteratoin, trying count:", k.Countsum)
		k.countNoNodeIterationChan <- 1
		return

	}

	//call alpha node to find the target id

	for index, _ := range (NodeToCall) {
		newNode := NodeToCall[index]
		go k.IterativeFindValue(newNode, k.ShortList.Target)
	}

}

func (k *Kademlia) DoIterativeFindNode(id ID) ([]Contact, error) {
	//find alpha nodes and generate short list
	err := k.ShortList.InitShortList(id)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , fn:InitShortList",err.Error())
	}
	err = k.LocalFindNode(id)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , fn:LocalFindNode",err.Error())
	}

	// if two iteration's closest node is same
	lastIteration :=false
	ActiveListLen := 0
	count := 0
	for ActiveListLen <= knum  && lastIteration == false  {

		// parallelism call alpha number of nodes

		go k.ParallelAlphaNode ()
		time.Sleep(300 * time.Millisecond)
		// check if closest node changed

		k.ShortList.TakeClosestChangeIn <- true
		closestChange := <- k.ShortList.TakeClosestChangeOut

		if closestChange {
			k.ShortList.ChangeClosestChan <- true
		}else {
			lastIteration = true
		}

		k.ShortList.TakeActiveListLenIn <- true
		ActiveListLen = <- k.ShortList.TakeActiveListLenOut
		k.TakeCountNoNodeInChan <- true
		count = <- k.TakeCountNoNodeOutChan

		for i := 0; i <ActiveListLen; i++ {
			println(k.ShortList.ActiveList[i].selfContact.NodeID.AsString())
		}

		if count > 5 {
			println("error: no more nodes found, less than k nodes returned, iteration find node exit!")
			k.CopySliceToShortList()
			return k.ShortList.shortList, errors.New("no more nodes found!!")

		}

	}

	println("activeListlen:", ActiveListLen)

	// if the number of nodes in shortlist less than k, call enough nonActive node to expand the shortlist
	if ActiveListLen < knum {
		numToCall := knum - ActiveListLen
		k.ShortList.TakeNonActiveListChanIn <- true
		nonactivelist := <- k.ShortList.TakeNonActiveListChanOut
		nonacLength := len(nonactivelist)
		if nonacLength >= numToCall {
			index := 0
			for ActiveListLen <= knum {
				if  index <  nonacLength{
					go k.IterativeCallNode(nonactivelist[index], id)
					index++
				}

				time.Sleep(100 * time.Millisecond)
				k.ShortList.TakeActiveListLenIn <- true
				ActiveListLen = <- k.ShortList.TakeActiveListLenOut
			}
		} else {
			println("no # of k nodes in network!!")
			k.CopySliceToShortList()
			return k.ShortList.shortList, errors.New("no more nodes found!!")

		}
	}


	//after the last iterative, sort the active node with distance to ID, move the active node into shortlist
	k.CopySliceToShortList()



	return k.ShortList.shortList, nil
}

func (k * Kademlia) CopySliceToShortList() {
	k.ShortList.TakeShortListChanIn <- true
	activelist := <- k.ShortList.TakeShortListChanOut
	for index,node := range(activelist) {
		if (index >= knum) {break}

		slContact := new(Contact)
		slContact.NodeID = node.selfContact.NodeID
		slContact.Port = node.selfContact.Port
		slContact.Host = node.selfContact.Host

		k.ShortList.shortList = append(k.ShortList.shortList, *slContact)
	}
}

func (k *Kademlia) DoIterativeStore(key ID, value []byte) ([]Contact,  error) {

	contacts, err := k.DoIterativeFindNode(key)

	if err!=nil {
		return nil, &CommandFailed{"FindNode failed"}
	}

	for _, node := range(contacts) {
		k.DoStore(&node, key, value)
	}

	return contacts, nil
}



func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {


	//find alpha nodes and generate short list
	err = k.ShortList.InitShortList(key)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , fn:InitShortList",err.Error())
	}
	err = k.LocalFindNode(key)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , fn:LocalFindNode",err.Error())
	}

	// if two iteration's closest node is same
	fmt.Println("leniterV",len(k.iterV))
	lastIteration :=false
	ActiveListLen := 0
	count := 0
	returnValue := make([]byte, 0)
	for ActiveListLen <= knum  && lastIteration == false && len(returnValue) == 0 {



		// parallelism call alpha number of nodes

		go k.ParallelAlphaNodeForValue ()
		time.Sleep(300 * time.Millisecond)
		// check if closest node changed

		k.ShortList.TakeClosestChangeIn <- true
		closestChange := <- k.ShortList.TakeClosestChangeOut

		if closestChange {
			k.ShortList.ChangeClosestChan <- true
		}else {
			lastIteration = true
		}

		k.ShortList.TakeActiveListLenIn <- true

		ActiveListLen = <- k.ShortList.TakeActiveListLenOut

		//fmt.Println("AC",ActiveListLen)
		//for _,node := range(k.ShortList.NonactiveList) {
		//	fmt.Println("NA",node.NodeID )
		//}

		k.TakeCountNoNodeInChan <- true
		//fmt.Println("Go active!")
		count = <- k.TakeCountNoNodeOutChan

		for i := 0; i <ActiveListLen; i++ {
			fmt.Println("Active List",k.ShortList.ActiveList[i].selfContact.NodeID)
		}

		if count > 5 {
			println("error: no more nodes found, less than k nodes returned, iteration find node exit!")
			k.CopySliceToShortList()
			return nil, &CommandFailed{k.ShortList.ClosestNode.NodeID.AsString()}

		}

		k.TakeItChanIn<-true
		returnValue = <- k.TakeItChanOut
		if len(returnValue) > 0 {
			println("i am here: ", string(returnValue[:]))
			return []byte(string(returnValue[:])), nil
		}

	}

	//return []byte(string(returnValue[:])), nil

	//fmt.Println("OMGactiveListlen:", ActiveListLen)
	//
	//
	if ActiveListLen < knum {
		numToCall := knum - ActiveListLen
		k.ShortList.TakeNonActiveListChanIn <- true
		nonactivelist := <- k.ShortList.TakeNonActiveListChanOut
		nonacLength := len(nonactivelist)
		if nonacLength >= numToCall {
			index := 0
			returnValue := make([]byte, 0)
			for ActiveListLen <= knum && len(returnValue) == 0{
				if  index <  nonacLength{
					go k.IterativeFindValue(nonactivelist[index],k.ShortList.Target)
					index++
				}

				time.Sleep(100 * time.Millisecond)
				k.ShortList.TakeActiveListLenIn <- true
				ActiveListLen = <- k.ShortList.TakeActiveListLenOut
				k.TakeItChanIn<-true
				returnValue = <- k.TakeItChanOut
				if len(returnValue) > 0 {
					println("i am here: ", string(returnValue[:]))
					return []byte(string(returnValue[:])), nil
				}
			}
		} else {
			println("no # of k nodes in network!!")
			k.CopySliceToShortList()
			return nil, &CommandFailed{k.ShortList.ClosestNode.NodeID.AsString()}

		}
	}


	//after the last iterative, sort the active node with distance to ID, move the active node into shortlist
	k.CopySliceToShortList()

	fmt.Println(len(k.ShortList.ActiveList))
	return nil, &CommandFailed{k.ShortList.ClosestNode.NodeID.AsString()}
}

// for ietrative_find_value return data
type vContact struct {
	returnContact  []Contact
	senderNodeID    ID
	senderHost      net.IP
	senderPort      uint16
	returnValue	[]byte
	SKey		ID
	dead		bool
}


func(k *Kademlia) IterativeFindValue(contact Contact, searchKey ID)  {

	hostname := contact.Host.String()
	portnum := strconv.Itoa(int(contact.Port))
	address := hostname + ":" + portnum

	client, err := rpc.DialHTTPPath("tcp", address, rpc.DefaultRPCPath+portnum)
	if err != nil {
		tmpNode := new(vContact)
		tmpNode.senderNodeID = contact.NodeID
		tmpNode.senderHost = contact.Host
		tmpNode.senderPort = contact.Port
		tmpNode.returnContact = nil
		tmpNode.returnValue = nil
		tmpNode.SKey = searchKey
		tmpNode.dead = true
		k.iterativeFindValueChan <- *tmpNode
	}


	//generate find value request and call the server FindValue function, receive respons(FindValueResult)
	FindValueReq := FindValueRequest{
		Sender : contact,
		MsgID  : NewRandomID(),
		Key    : searchKey,
	}

	var FindValueRes FindValueResult

	err = client.Call("KademliaRPC.FindValue", FindValueReq, &FindValueRes)

	//update bucket if the response is not null
	if FindValueRes.Nodes != nil {
		for index := 0; index < len(FindValueRes.Nodes); index++ {
			k.ContactUpdateChan <- &FindValueRes.Nodes[index]
		}

		tmpNode := new(vContact)
		tmpNode.senderNodeID = contact.NodeID
		tmpNode.senderHost = contact.Host
		tmpNode.senderPort = contact.Port
		tmpNode.returnContact = FindValueRes.Nodes
		tmpNode.returnValue = nil
		tmpNode.SKey = searchKey
		tmpNode.dead = false
		k.iterativeFindValueChan <- *tmpNode
	}else{
		tmpNode := new(vContact)
		tmpNode.senderNodeID = contact.NodeID
		tmpNode.senderHost = contact.Host
		tmpNode.senderPort = contact.Port
		tmpNode.returnContact = nil
		tmpNode.returnValue = FindValueRes.Value
		tmpNode.SKey = searchKey
		tmpNode.dead = false
		k.iterativeFindValueChan <- *tmpNode
	}



	//return  FindValueRes.Nodes, nil
}

// For project 3!
func (k *Kademlia) Vanish(data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanashingDataObject) {
	return
}

func (k *Kademlia) Unvanish(searchKey ID) (data []byte) {
	return nil
}
