// Class RouterTable
// Author: Shixin Luo

package libkademlia

import (
	"net/rpc"
	"strconv"
	"container/list"

	//"sort"
	//"fmt"
	//"fmt"

)



type RoutingTable struct {
	SelfContact		Contact 				// Node Identifier
	KBuckets 		[IDBits]*list.List;		// Kademlia routing table -- KBuckets
}

// Return new Routing Table
func NewRoutingTable(contact Contact) (table *RoutingTable) {
  table = new(RoutingTable)
  table.SelfContact = contact
  for i := 0; i < IDBits; i++ {
    table.KBuckets[i] = list.New();   //create new kbucket
  }
  return;
}

// Update routing table based on id of node communicated with
func (table *RoutingTable) Update(contact *Contact) {
	prefix_len := table.SelfContact.NodeID.Xor(contact.NodeID).PrefixLen()
	//fmt.Println(contact.NodeID,"contactNOdeID")
	//fmt.Println(table.SelfContact.NodeID,"table")
	//fmt.Println(prefix_len,"prefix_len")

	if prefix_len != 160 {
		l := table.KBuckets[prefix_len]
		isFound := false					// Search Flag
		var elementFound *list.Element 		// Search Result
		// Iterate over list to find out if given contact exists
		for e := l.Front(); e != nil; e = e.Next() {
			if e.Value.(Contact).NodeID.Equals(contact.NodeID) {
				isFound = true
				elementFound = e
				break
			}
		}
		//fmt.Println("yes",l)
		if isFound {							// If Node already exists, move it to the tail
			//fmt.Println("FOund")
			l.MoveToBack(elementFound)
		} else {								// If not exists, check if list is full
			if l.Len() < knum {					// If list is not full, append new Node to the tail
				l.PushBack(*contact)
				//fmt.Println("pushing")
			} else {							// If list is full, check activity of first Node
				if table.CheckActivity(l) {		// If the first Node is active, move it to the tail
					l.MoveToBack(l.Front())
					//fmt.Println("active")
				} else {
					//fmt.Println("not active")// If the first Node is inactive, delete it and append new Node to the tail
					l.Remove(l.Front())
					l.PushBack(*contact)
				}
			}
		}
	}
}

// Check if first Node in the list is active
func (table *RoutingTable) CheckActivity(l *list.List) bool {
	ping := PingMessage{table.SelfContact, NewRandomID()}
	var pong PongMessage

	port_str := strconv.Itoa(int(l.Front().Value.(Contact).Port))
	host := l.Front().Value.(Contact).Host
	port := l.Front().Value.(Contact).Port
	address := host.String() + ":" + strconv.FormatInt(int64(port), 10)
	client, err := rpc.DialHTTPPath("tcp", address, rpc.DefaultRPCPath+port_str)
	if err != nil {
		return false
	}
	err = client.Call("KademliaRPC.Ping", ping, &pong)
	if err != nil {
		return false
	} else {
		return true
	}

	client.Close()

	return false
}

// Entry of closest node list
type ContactEntry struct {
	Contact 	Contact
	Distance 	ID

}

func copyToSlice(start, end *list.Element, sortSlice []ContactEntry, destNodeID ID) []ContactEntry {
	for e := start; e != end; e = e.Next() {
    		contact := e.Value.(Contact)
		distance := contact.NodeID.Xor(destNodeID)
		sortSlice = append(sortSlice, ContactEntry{contact, distance})
  	}
	return sortSlice
}

// Find closest k nodes, return the contact slice
func (table *RoutingTable) FindClosest(destNodeID ID, num int) (ret []Contact) {
  // Create slice to store nodes' contacts
  ret = make([]Contact, 0)
  // Temporary slice to store raw search results
  sortSlice := make([]ContactEntry, 0)

  prefix_len := destNodeID.Xor(table.SelfContact.NodeID).PrefixLen();
  l := table.KBuckets[prefix_len];

  // Copy found bucket list to the slice
  sortSlice = copyToSlice(l.Front(), nil, sortSlice, destNodeID);

  // Fill slice with contents of prefix-adjecent lists
  for i := 1; (prefix_len-i >= 0 || prefix_len+i < IDBits) && len(sortSlice) < num; i++ {
    if prefix_len - i >= 0 {
      l = table.KBuckets[prefix_len - i];
      sortSlice = copyToSlice(l.Front(), nil, sortSlice, destNodeID);
    }
    if prefix_len + i < IDBits {
      l = table.KBuckets[prefix_len + i];
      sortSlice = copyToSlice(l.Front(), nil, sortSlice, destNodeID);
    }
  }
  
  // Sort Slice in distance ascending order
   for i := 0; i < len(sortSlice); i ++ {
	for j := i+1; j < len(sortSlice); j++ {
		if sortSlice[i].Distance.Compare(sortSlice[j].Distance) == 1 {
			sortSlice [i], sortSlice [j] = sortSlice [j], sortSlice [i]
		}
	}
   }

  // Cut Slice into given size
  if len(sortSlice) > num {
    sortSlice = sortSlice[:num]
  }

  //fmt.Println(sortSlice)
  for j := 0; j < len(sortSlice); j ++ {
	  ret = append(ret, sortSlice[j].Contact)
  }
  return;
}

func (table *RoutingTable) FindContact(nodeID ID)(ret Contact) {
	prefix_len := table.SelfContact.NodeID.Xor(nodeID).PrefixLen()
	l := table.KBuckets[prefix_len]
	for e := l.Front(); e != nil; e = e.Next() {
		if e.Value.(Contact).NodeID.Equals(nodeID) {
			return e.Value.(Contact)
		}
	}
	return
}