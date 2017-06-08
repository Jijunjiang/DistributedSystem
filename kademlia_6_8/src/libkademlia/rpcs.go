package libkademlia

// Contains definitions mirroring the Kademlia spec. You will need to stick
// strictly to these to be compatible with the reference implementation and
// other groups' code.

import (
	"net"

	//"errors"
	//"fmt"
)

type KademliaRPC struct {
	kademlia *Kademlia
}

// Host identification.
type Contact struct {
	NodeID ID
	Host   net.IP
	Port   uint16
}

///////////////////////////////////////////////////////////////////////////////
// PING
///////////////////////////////////////////////////////////////////////////////
type PingMessage struct {
	Sender Contact
	MsgID  ID
}

type PongMessage struct {
	MsgID  ID
	Sender Contact
}

func (k *KademliaRPC) Ping(ping PingMessage, pong *PongMessage) error {
	// TODO: Finish implementation
	pong.MsgID = CopyID(ping.MsgID)
	// Specify the sender
	pong.Sender = k.kademlia.SelfContact
	// Update contact, etc
	k.kademlia.ContactUpdateChan <- &ping.Sender
	
	return nil
}

///////////////////////////////////////////////////////////////////////////////
// STORE
///////////////////////////////////////////////////////////////////////////////
type StoreRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
	Value  []byte
}

type StoreResult struct {
	MsgID ID
	OperationInfo string
	Err   error

}

func (k *KademliaRPC) Store(req StoreRequest, res *StoreResult) error {

	/* receive store request, reply store result */
 	IncomeData := &Data{
 		Key : req.Key,
 		Value : req.Value,
 	}

	k.kademlia.DataStoreChan <- IncomeData
	popInfo := <-k.kademlia.DataResChan

	if popInfo == true {
		res.MsgID =  CopyID(req.MsgID)
		res.Err = nil
		res.OperationInfo = "success"
	}else{
		res.MsgID =  CopyID(req.MsgID)
		res.Err = nil
		res.OperationInfo = "failed"
	}







	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_NODE
///////////////////////////////////////////////////////////////////////////////
type FindNodeRequest struct {
	Sender Contact
	MsgID  ID
	NodeID ID
}

type FindNodeResult struct {
	MsgID ID
	Nodes []Contact
	Err   error
}

func (k *KademliaRPC) FindNode(req FindNodeRequest, res *FindNodeResult) error {

	/* receive findNoderequest reply FindNodeResult */

	// trasfer to channel handle and search node

	k.kademlia.NodeSearchInChan <- req.NodeID
	nodeResult := <- k.kademlia.NodeSearchOutChan

	if nodeResult != nil {
		res.MsgID = CopyID(req.MsgID)
		res.Nodes = nodeResult
		res.Err = nil
	} else {
		res.MsgID = CopyID(req.MsgID)
		res.Nodes = nil
		res.Err = nil

	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////
// FIND_VALUE
///////////////////////////////////////////////////////////////////////////////
type FindValueRequest struct {
	Sender Contact
	MsgID  ID
	Key    ID
}

// If Value is nil, it should be ignored, and Nodes means the same as in a
// FindNodeResult.
type FindValueResult struct {
	MsgID ID
	Value []byte
	Nodes []Contact
	Err   error
}

func (k *KademliaRPC) FindValue(req FindValueRequest, res *FindValueResult) error {

	/* RPC server receive FindValueRequests and reply FindValue Result */
	k.kademlia.ValueSearchInChan <- req.Key
	returnValue := <- k.kademlia.ValueResultChan


	// if the value is found in the RPC target Node, return the value
	//else return the k(20) closest node
	if returnValue !=nil {

		res.MsgID = CopyID(req.MsgID)
		res.Value = returnValue
		res.Nodes = nil
		res.Err = nil


	}else{

		k.kademlia.ValueSearchNodeChan <- req.Sender.NodeID
		returnNodes := <- k.kademlia.ValueNodeResult

		if returnNodes != nil {
			res.MsgID = CopyID(req.MsgID)
			res.Value = nil
			res.Nodes = returnNodes
			res.Err = nil

		}else{
			res.MsgID = CopyID(req.MsgID)
			res.Value = nil
			res.Nodes = nil
			res.Err =   nil

		}
	}




	return nil
}


//project2
///////////////////////////////////////////////////////////////////////////////
// DoIterativeFindNode
///////////////////////////////////////////////////////////////////////////////
type SList struct {
	ActiveList 		[]ContactWithDistance
	NonactiveList 		[]Contact
	ClosestNode 		Contact
	ClosestDistance		ID
	Target			ID
	ExpandShortListChan	chan []Contact
	MoveToActiveChan	chan Contact
	DeleteChan		chan int
	DeleteDoneChan        chan bool
	//SortActiveListChan	chan []ContactWithDistance
	//SortReturnChan		chan []ContactWithDistance
	TakeNonActiveListChanOut	chan []Contact
	TakeNonActiveListChanIn	chan bool
	TakeActiveListLenIn	chan bool
	TakeActiveListLenOut	chan int

	TakeClosestChangeIn		chan bool
	TakeClosestChangeOut		chan bool

	ChangeClosestChan	chan bool

	TakeShortListChanIn	chan bool
	TakeShortListChanOut	chan []ContactWithDistance

	contactsVisited 	map[ID] bool
	closestNodeChange       bool
	shortList		[]Contact

}

type ContactWithDistance struct {
	selfContact Contact
	Distance	ID
}

func (s *SList) ShortListChannelHandle () {
	for {
		select {
			case listToExpand := <-s.ExpandShortListChan:
				s.ExpandShortList(listToExpand)

			case node := <-s.MoveToActiveChan:
				s.MoveToActive(node)
				//s.TakeActiveListLenIn <- true

			case preLen := <-s.DeleteChan:
				//delete the preLen number of data in NonActiveList
				s.NonactiveList = s.NonactiveList[preLen:]
				s.DeleteDoneChan <- true

			case out := <- s.TakeNonActiveListChanIn:
				if(out) {
					s.TakeNonActiveListChanOut <- s.NonactiveList
				}

			case takeclosest := <- s.TakeClosestChangeIn:
				if (takeclosest) {
					s.TakeClosestChangeOut <- s.closestNodeChange
				}
			case closestChange := <-s.ChangeClosestChan:
				s.closestNodeChange = closestChange

			case	takeActiveLen := <- s.TakeActiveListLenIn:
				if (takeActiveLen) {
					s.TakeActiveListLenOut <- len(s.ActiveList)
				}
			case	takeShortList := <- s.TakeShortListChanIn:
				if (takeShortList) {
					s.TakeShortListChanOut <- s.ActiveList
				}

		}

	}
}

func (s *SList) InitShortList(target ID) error {
	s.Target = target
	for i := 0; i < IDBytes; i++ {
		s.ClosestDistance[i] = uint8(255)
	}
	s.ActiveList = make([]ContactWithDistance,0)
	s.NonactiveList = make([]Contact,0)

	s.ExpandShortListChan = make(chan []Contact)
	s.MoveToActiveChan = make(chan Contact)
	s.DeleteChan = make(chan int)
	s.DeleteDoneChan = make( chan bool)
	//s.SortActiveListChan = make (chan []ContactWithDistance)
	//s.SortReturnChan = make(chan []ContactWithDistance)
	s.TakeNonActiveListChanIn = make(chan bool)

	s.TakeNonActiveListChanOut = make(chan []Contact)
	s.contactsVisited = make(map[ID] bool)


	s.TakeClosestChangeIn = make (chan bool)
	s.TakeClosestChangeOut = make (chan bool)
	s.ChangeClosestChan = make(chan bool)

	s.TakeActiveListLenIn = make (chan bool)
	s.TakeActiveListLenOut = make (chan int)

	s.TakeShortListChanIn = make(chan bool)
	s.TakeShortListChanOut = make(chan []ContactWithDistance)

	s.closestNodeChange = true
	s.shortList = make([]Contact,0)
	go s.ShortListChannelHandle()
	return nil
}


func (s *SList) MoveToActive(contact Contact) error {

	//if (!s.contactsVisited[contact.NodeID]){
	tmpContact := new(ContactWithDistance)
	tmpContact.selfContact.NodeID = contact.NodeID
	tmpContact.selfContact.Port = contact.Port
	tmpContact.selfContact.Host = contact.Host
	tmpContact.Distance = contact.NodeID.Xor(s.Target)
	s.ActiveList = append(s.ActiveList,*tmpContact)
	//}
	return nil
}

func (s *SList) ExpandShortList(contact []Contact) error {
	// add the new returned node from Iterative RPC to the Nonactive Node list
	// if the closest node change, set flag closestNodeChange to true
	tempDistance := s.ClosestDistance

	// find the new closest node
	for i := 0; i < len(contact); i++{

		if (!s.contactsVisited[contact[i].NodeID]){
			s.NonactiveList = append(s.NonactiveList, contact[i])
			distance := contact[i].NodeID.Xor(s.Target)
			s.contactsVisited[contact[i].NodeID] = true
			if (distance.Less(s.ClosestDistance)) {
				s.ClosestDistance = distance
				s.ClosestNode = contact[i]
				s.closestNodeChange = true
			}
		}
	}

	if (s.ClosestDistance.Compare( tempDistance) == 0) {
		s.closestNodeChange = false
	}

	// sort the nonactive list for catching the closest nodes to call as active node
	for i := 0; i < len(s.NonactiveList); i ++ {
		for j := i+1; j < len(s.NonactiveList); j++ {
			distance1 := s.NonactiveList[i].NodeID.Xor(s.Target)
			distance2 := s.NonactiveList[j].NodeID.Xor(s.Target)
			if distance1.Compare(distance2) != -1 {
				s.NonactiveList[i], s.NonactiveList[j] = s.NonactiveList[j], s.NonactiveList[i]
			}
		}
	}




	return nil
}


////////////////////////////////////////////////////////////////////////////////////
// For Project 3

type GetVDORequest struct {
	Sender Contact
	VdoID  ID
	MsgID  ID
}

type GetVDOResult struct {
	MsgID ID
	VDO   VanishingDataObject
}

func (k *KademliaRPC) GetVDO(req GetVDORequest, res *GetVDOResult) error {
	// TODO: Implement.
	k.kademlia.passVDOinChan <-req.VdoID
	VDOres := <- k.kademlia.passVDOoutChan
	if VDOres.AccessKey != 0 {
		res.MsgID = req.MsgID
		res.VDO = VDOres
	}else{
		res.MsgID = req.MsgID
		res.VDO = VDOres
	}
	return nil
}


