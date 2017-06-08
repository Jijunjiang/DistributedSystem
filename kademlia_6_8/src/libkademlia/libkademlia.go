package libkademlia

// Contains the core kademlia type. In addition to core state, this type serves
// as a receiver for the RPC methods, which is required by that package.

import (
	//"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"container/list"
	"os"
	"time"

	"errors"
	//"fmt"
	"fmt"
)

const (
	alpha = 3
	b     = 8 * IDBytes
	knum  = 20
	//TODO change here to test timeout, in our test case it is set 50 seconds
	epochSeconds = 3600 * 8      // the epoch time is in seconds, default 8 hours (3600 * 8 seconds)
	//TODO change here to test timeout, in our test case it is set 15 seconds
	refreshEpochOffset = 600	// the offset time to start refreshing the locations before the epoches
					// used to make sure clients unvanish the right data

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
	DataResChan		chan bool				//signifying the operation
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
	DataFlagChan 	   chan int
////////////////////////////////////////////////project2/////////////////////////////

	ShortList	SList
	iterativeFindResChan   chan sContact
	iterativeFindValueChan  chan vContact
	iterativeFindValueResInChan chan int
	iterativeFindValueResOutChan chan []byte
	countNoNodeIterationChan	chan int
	TakeCountNoNodeInChan		chan bool
	TakeCountNoNodeOutChan		chan int
	iterV				[]byte
	deleteRequestChan	chan int
	deleteOfferChan		chan bool
//////////////////////////////////////////////////////////////////////////////////////
	VDOmap			map[ID]VanishingDataObject
	passVDOinChan		chan ID
	passVDOoutChan		chan VanishingDataObject
	vanishGetResInChan 	chan VanishingDataObject
	vanishGetResOutChan 	chan []byte
	VDOtempStoreChan	chan VanishingDataObject
	ReallyFindVDOchan	chan VanishingDataObject


	VanishTimeOutStartChan	chan int
	VanishTimeOutVDOChan	chan VanishingDataObject
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
	k.DataResChan = make(chan bool)

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
	k.DataFlagChan = make(chan int)
///////////////////////////////////////////////project2/////////////////////////////

	k.iterativeFindResChan = make(chan sContact)
	k.iterativeFindValueChan = make(chan vContact)
	k.iterativeFindValueResInChan =  make(chan int)
	k.iterativeFindValueResOutChan = make(chan []byte)
	k.countNoNodeIterationChan = make(chan int)
	k.TakeCountNoNodeInChan = make(chan bool)
	k.TakeCountNoNodeOutChan = make(chan int)
	k.iterV = make([]byte, 0)
	k.deleteRequestChan = make(chan int)
	k.deleteOfferChan = make(chan bool)

////////////////////////////////////////////////////////////////////////////////////
	k.VDOmap = make(map[ID]VanishingDataObject)
	k.passVDOinChan = make(chan ID)
	k.passVDOoutChan =	make(chan VanishingDataObject)
	k.vanishGetResInChan = 	make(chan VanishingDataObject)
	k.vanishGetResOutChan 	=make(chan []byte)
	k.VDOtempStoreChan = make(chan VanishingDataObject)
	k.ReallyFindVDOchan = make(chan VanishingDataObject)

	k.VanishTimeOutStartChan = make(chan int)
	k.VanishTimeOutVDOChan = make(chan VanishingDataObject)

///////////////////////////////////////////////////////////////////////////////////
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
				//k.DataResChan <- false
				//k.DataFlagChan <- 0

				k.DataMap[data.Key] = data.Value
				k.DataResChan <- true
			} else {
				k.DataMap[data.Key] = data.Value
				k.DataResChan <- true
				//k.DataFlagChan <- 1
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
			if contactFound.dead == false {
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
					//fmt.Println("Really found!")
					//fmt.Println(contactFound.returnValue)
					//toStoreNode := k.Table.FindClosest(k.SelfContact.NodeID,2)
					//fmt.Println("shit")
					//fmt.Println("qqq",k.ShortList.ClosestNode.NodeID)
					err := k.DoStore(&k.ShortList.ClosestNode, contactFound.SKey, contactFound.returnValue)

					if err != nil {

						ret := k.ShortList.ActiveList

						//for _,node := range(ret) {
						//	fmt.Println("In AC",node.selfContact.NodeID)
						//}

						for i := 0; i < len(ret); i ++ {
							for j := i + 1; j < len(ret); j++ {
								if ret[i].Distance.Compare(ret[j].Distance) == 1 {
									ret [i], ret [j] = ret [j], ret [i]
								}
							}
						}

						//fmt.Println("ret", ret)

						for _, node := range (ret) {
							//fmt.Println("to stoeeeeee",node.selfContact.NodeID)
							tmpNode := new(Contact)
							tmpNode.NodeID = node.selfContact.NodeID
							tmpNode.Port = node.selfContact.Port
							tmpNode.Host = node.selfContact.Host
							err1 := k.DoStore(tmpNode, contactFound.SKey, contactFound.returnValue)
							if err1 == nil {
								//fmt.Println("stored node",tmpNode.NodeID)
								break
							}
						}
					}

					k.iterV = contactFound.returnValue
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
		case takeValue := <-k.iterativeFindValueResInChan:
			if takeValue == 1 {
				k.iterativeFindValueResOutChan <- k.iterV
			}

		case deletePre := <-k.deleteRequestChan:
			k.ShortList.DeleteChan <- deletePre
			deleteDone := <- k.ShortList.DeleteDoneChan
			k.deleteOfferChan <- deleteDone


			////////////for vanish//////////////////////
		case vanishObjectID := <- k.passVDOinChan:
			_, ok := k.VDOmap[vanishObjectID]
			var VDORes VanishingDataObject
			if ok {
				VDORes = k.VDOmap[vanishObjectID]
			} else {
				VDORes.AccessKey = 0
				VDORes.Ciphertext = nil
				VDORes.NumberKeys = byte(0)
				VDORes.Threshold = byte(0)
			}

			k.passVDOoutChan <- VDORes
		case VDOFindResult := <- k.VDOtempStoreChan:
			k.ReallyFindVDOchan <- VDOFindResult
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
	//fmt.Println(storeRes.OperationInfo)
	//fmt.Println(storeRes.Err)

	if storeRes.OperationInfo == "failed" {
		fmt.Printf("to iverwrite!")
		return &CommandFailed{"Not implemented"}
	}else {
		return nil
	}

	//
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
		//println("dead",contact.NodeID.AsString())
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
		k.deleteRequestChan <- alpha
		_ = <- k.deleteOfferChan


	} else if length > 0{

		NodeToCall = nonactivelist[:]

		k.deleteRequestChan <- len(nonactivelist)
		_= <- k.deleteOfferChan

	} else {
		//println("no node to do iteratoin, trying count:", k.Countsum)
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
		k.deleteRequestChan <- alpha
		_= <- k.deleteOfferChan

		//fmt.Println("delete",1)

	} else if length > 0{

		NodeToCall = nonactivelist[:]

		k.deleteRequestChan <- len(nonactivelist)
		_= <- k.deleteOfferChan

		//fmt.Println("delete",2)
	} else {
		//println("no node to do iteratoin, trying count:", k.Countsum)
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
	k.iterV = make([]byte,0)
	err := k.ShortList.InitShortList(id)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , fn:InitShortList",err.Error())
	}
	k.ShortList.contactsVisited[k.SelfContact.NodeID] = true
	err = k.LocalFindNode(id)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , fn:LocalFindNode",err.Error())
	}

	// if two iteration's closest node is same
	lastIteration :=false
	ActiveListLen := 0
	count := 0

	// if enough nodes in activelist OR the closest node not change break the iteration

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
		if count > 5 {
			//println("error: no more nodes found, less than k nodes returned, iteration find node exit!")
			k.CopySliceToShortList()
			return k.ShortList.shortList, errors.New("no more nodes found!!")
		}
	}

	// if the number of nodes in shortlist less than k, call enough nonActive node to expand the shortlist
	// (if no enough nodes in the nonactive list already collected, iterative call the nodes in nonactive list
	// and return as many nodes as possible)
	if ActiveListLen < knum {
		k.ShortList.TakeNonActiveListChanIn <- true
		nonactivelist := <- k.ShortList.TakeNonActiveListChanOut
		nonacLength := len(nonactivelist)

		for  nonacLength > 0 && ActiveListLen <= knum {
			index := 0
			for index < nonacLength {
				go k.IterativeCallNode(nonactivelist[index], id)
				time.Sleep(10 * time.Millisecond)
				index++
			}
			k.deleteRequestChan <- len(nonactivelist)

			_=<- k.deleteOfferChan

			k.ShortList.TakeNonActiveListChanIn <- true
			nonactivelist = <- k.ShortList.TakeNonActiveListChanOut
			nonacLength = len(nonactivelist)

			k.ShortList.TakeActiveListLenIn <- true
			ActiveListLen = <- k.ShortList.TakeActiveListLenOut
		}

		println("less than k nodes in network!!")
		k.CopySliceToShortList()
		return k.ShortList.shortList, nil
	}

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
	k.iterV = make([]byte,0)
	contacts, err := k.DoIterativeFindNode(key)
	if err!=nil {
		return nil, &CommandFailed{"FindNode failed"}
	}
	for _, node := range(contacts) {
		k.DoStore(&node, key, value)
		//fmt.Println(node.NodeID)
	}
	return contacts, nil
}

func (k *Kademlia) DoIterativeFindValue(key ID) (value []byte, err error) {
	k.iterV = make([]byte,0)
	err = k.ShortList.InitShortList(key)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , fn:InitShortList",err.Error())
	}
	k.ShortList.contactsVisited[k.SelfContact.NodeID] = true
	err = k.LocalFindNode(key)
	if err != nil {
		fmt.Fprintf(os.Stderr,"Fatal error: %s , fn:LocalFindNode",err.Error())
	}
	lastIteration :=false
	ActiveListLen := 0
	count := 0
	returnValue := make([]byte,0)
	for ActiveListLen <= knum  && lastIteration == false && len(returnValue)==0 {

		go k.ParallelAlphaNodeForValue ()
		time.Sleep(300 * time.Millisecond)
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

		if count > 5 {
			//println("error: no more nodes found, less than k nodes returned, iteration find node exit!")
			k.CopySliceToShortList()
			return nil, &CommandFailed{k.ShortList.ClosestNode.NodeID.AsString()}

		}

		k.iterativeFindValueResInChan <- 1

		returnValue = <-k.iterativeFindValueResOutChan
		if len(returnValue) >0 {
			return returnValue,nil
		}


	}
	if ActiveListLen < knum {

		k.ShortList.TakeNonActiveListChanIn <- true
		nonactivelist := <- k.ShortList.TakeNonActiveListChanOut
		nonacLength := len(nonactivelist)
		index := 0
		for  nonacLength > 0 && ActiveListLen <= knum {
			index = 0
			for index < nonacLength {
				go k.IterativeFindValue(nonactivelist[index], key)
				time.Sleep(10 * time.Millisecond)
				index++
			}
			k.deleteRequestChan <- len(nonactivelist)
			_=<- k.deleteOfferChan
			//time.Sleep(50 * time.Millisecond)
			k.ShortList.TakeNonActiveListChanIn <- true
			nonactivelist = <- k.ShortList.TakeNonActiveListChanOut
			nonacLength = len(nonactivelist)
			k.ShortList.TakeActiveListLenIn <- true
			ActiveListLen = <- k.ShortList.TakeActiveListLenOut
			k.iterativeFindValueResInChan <- 1

			returnValue = <-k.iterativeFindValueResOutChan
			if len(returnValue) > 0 {
				return returnValue,nil
			}

		}
		k.CopySliceToShortList()
		return nil, &CommandFailed{k.ShortList.ClosestNode.NodeID.AsString()}
	}

	k.CopySliceToShortList()
	return nil, &CommandFailed{k.ShortList.ClosestNode.NodeID.AsString()}

}

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

	FindValueReq := FindValueRequest{
		Sender : contact,
		MsgID  : NewRandomID(),
		Key    : searchKey,
	}

	var FindValueRes FindValueResult

	err = client.Call("KademliaRPC.FindValue", FindValueReq, &FindValueRes)

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

}


// For project 3!
func (k *Kademlia) Vanish(vdoID ID, data []byte, numberKeys byte,
	threshold byte, timeoutSeconds int) (vdo VanishingDataObject) {
	res, err := k.VanishData(vdoID, data,numberKeys,threshold,timeoutSeconds, k.getEpoch())
	if err != nil{
		fmt.Fprintf(os.Stderr,"Fatal error: %s , Vanish Error",err.Error())
	}
	go k.UpdateTimeOutVdo(timeoutSeconds)

	k.VanishTimeOutVDOChan <- res
	TimeNow := time.Now().UTC()
	startTime := int(TimeNow.Weekday())*24*3600 + int(TimeNow.Hour())*3600 +
		int(TimeNow.Minute())*60 + int(TimeNow.Second())
	k.VanishTimeOutStartChan <- startTime
	return res
}

func (k *Kademlia) Unvanish(nodeID ID, searchKey ID) (data []byte) {


	if nodeID.Equals(k.SelfContact.NodeID) {
		//fmt.Println("hi")
		localVDO,ok := k.VDOmap[searchKey]

		if ok {
			k.VDOtempStoreChan <- localVDO
		}else{
			nodeSet, err := k.DoIterativeFindNode(nodeID)

			if err != nil {
				fmt.Fprintf(os.Stderr, "Fatal error: %s , IFindNodeErrorInUnvanish", err.Error())
			}
			for _, node := range (nodeSet) {
				go k.findVDOIterative(node, searchKey)

			}
		}
		

	}else {
		nodeSet, err := k.DoIterativeFindNode(nodeID)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Fatal error: %s , IFindNodeErrorInUnvanish", err.Error())
		}
		for _, node := range (nodeSet) {
			go k.findVDOIterative(node, searchKey)

		}
	}
	VDOFinRes := <-k.ReallyFindVDOchan
	data, _ = k.UnvanishData(VDOFinRes)
	return data
}

func (k *Kademlia) findVDOIterative(contact Contact, VDOid ID) error{

	//build RPC connection
		hostname := contact.Host.String()
		portnum := strconv.Itoa(int(contact.Port))
		address := hostname + ":" + portnum

		//generate ping, call server and receive pong
		reqVDO := &GetVDORequest{
			Sender : k.SelfContact,
			MsgID : NewRandomID(),
			VdoID: VDOid,
		}
		var resVDO GetVDOResult
		client, err := rpc.DialHTTPPath("tcp", address, rpc.DefaultRPCPath+portnum)
		if err != nil {
			fmt.Fprintf(os.Stderr,"Fatal error: %s , findVDOIterativeError",err.Error())
		}
		//call other node's rpc function ping

		err = client.Call("KademliaRPC.GetVDO", reqVDO, &resVDO)


		if err != nil {
			fmt.Fprintf(os.Stderr,"Fatal error: %s , findVDOIterativeError",err.Error())
		}

		// no error update bucket and return

		if resVDO.VDO.AccessKey != 0 {
			k.VDOtempStoreChan <- resVDO.VDO
		}
		//client.Close()
		return nil


}
