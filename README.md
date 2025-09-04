# D7024E Lab Assignment: Creating a Peer-to-Peer Distributed Data Store
This description contains information that outlines the lab element in the D7024E ([Mobile and Distributed Computing Systems](https://www.ltu.se/en/education/syllabuses/course-syllabus?id=D7024E)) course at [Luleå Technical University](https://www.ltu.se/).

This description is divided into two parts. The first, [Lab Specification](#lab-specification), outlines the requirements of the lab module. The second, [Appendices](#appendices), includes some tips that you might find useful in order to complete the lab.

This description is a work in progress. Please report any errors you find.

# Lab Specification

## Introduction
The objective of this assignment is to produce a working [*Distributed Data Store* (DDS)](https://en.wikipedia.org/wiki/Distributed_data_store). 
In contrast to a traditional database, a DDS stores its *data objects* on many different computers rather than only one.
Here, we consider a *data object* to be an array of bytes with a well-known structure, such as a UTF-8 text or JSON. 
These computers together make up a single *storage network*, which internally keeps track of what objects are stored and which *nodes* keep copies of them. 
A *node*, or *peer*, is simply a participant in a peer-to-peer network. 
The system you are to create will be able to store and locate data objects by their [*hashes*](https://en.wikipedia.org/wiki/Hash_function) in such a storage network, which is formed by many running instances of your system.

The primary intention of the lab is to help you understand, both practically and theoretically, how modern distributed applications are built and some design challenges faced while constructing them.
We want you to see for yourself how these are more complicated and less efficient than their traditional client-server counterparts, but also how they facilitate extraordinary degrees of scalability, fault tolerance and concurrency.

We also want to introduce you to some of the tools and technologies that are commonly employed to build and manage distributed systems.
In particular, you are advised to write the program in [Google's Go language](https://tour.golang.org/), which was specifically designed for writing distributed applications, and for managing nodes by using [Docker containers](https://docs.docker.com/get-started/).
You are not forced to use either, but you will not be able to receive programming or container help from the teaching assistants if you choose to use any other programming language or container solution.
In other words, you are required to use a programming language and a container solution, but you are not strictly required to use Go or Docker.
If you decide to use Go, we provide code you can use as a starting point in [GitHub](kademlia/).
We would **highly** recommend using Go as your programming language of choice.
From experience, using any other language has historically been a struggle for students, as even if you choose another programming language, you must adhere to the exact same requirements.

Lastly, we want to give you a gentle introduction to [*Agile Software Development*](https://en.wikipedia.org/wiki/Agile_software_development) (ASD).
You will not be assessed on any aspect of ASD, but we use some of its terminology in this lab description.
Many of you, if not most, who will end up as software engineers after graduation will work according to this methodology, or at least a variant of it.
Using it to complete the lab can be an excellent way for you to coordinate your programming efforts.

*Best of luck,*  
*Teachers and Teaching Assistants*

## Objectives
To pass the lab assignment, you and your group members must create an application that fulfils the *mandatory objectives*, listed below.
Completing only those objectives will give you the lowest passing grade for the lab if no serious bugs can be observed.
To increase your grade, you must complete the *qualifying objectives* listed after the mandatory ones.
Your lab grade will depend on the number of points you acquire after having completed all mandatory objectives.
The number of points you get for each objective is stated within square brackets after its name.

Note that during assessment, each group member is expected to be able to demonstrate every objective, as well as describe how and why you approached it the way you did.
There are also further requirements that must be considered and delimitations we make to simplify the lab a bit.

### Mandatory
**M1**: *Network formation*. **[5p]**.
Your nodes must be able to form networks as described in the [Kademlia paper](kademlia-description.pdf)[^1]. Kademlia is a protocol for facilitating *[Distributed Hash Tables](https://en.wikipedia.org/wiki/Distributed_hash_table)* (DHTs). Concretely, the following aspects of the algorithm must be implemented:
1. **Pinging**: This means that you must implement and use the `PING` message.
2. **Network joining**: Given the IP address and any other data you decide, any single node must be able to join or form a network with the other node.
3. **Node lookup**: When part of a network, each node must be able to retrieve the contact information of any other node in the same network.

**M2**: *Object Distribution*. **[5p]**.
The network your nodes form must be able to manage the distribution, storage and retrieval of data objects, as described in the Kademlia paper. Note that in the Kademlia paper, objects are referred to as *values* and their hashes as *keys*.
Concretely, you must implement the following aspects of Kademlia:
1. **Storing objects**: When part of a network, it must be possible for any node to upload an object that will end up at the designated storage nodes. In Kademlia terminology, the *designated nodes* are the *K* nodes nearest to the hash of the data object in question.
2. **Finding objects**: When part of a network with uploaded objects, it must be possible to find and download any object, as long as it is stored by at least one designated node.

**M3**: *Command line interface*. **[5p]**.
Each node must provide a command line interface through which the following commands can be executed:
1. `put`: Takes a single argument, the contents of the file you are uploading, and outputs the hash of the object, if it can be uploaded successfully.
2. `get`: Takes a hash as its only argument, and outputs the contents of the object and the node it was retrieved from, if it could be downloaded successfully.
3. `exit`: Terminates the node.

**M4**: **Unit testing**. **[5p]**.
You must demonstrate that the core parts of your implementation work as expected by writing unit tests.
Unit tests check that the internal constructs of an application behave as expected, such as that the calculation of `XOR` distances is correct, or that contacts are inserted at the correct places in buckets, and so on.
Your unit tests must include some type of **network emulation** where you can emulate **at least 1000 nodes**, and **you must include some type of package dropping functionality**.
Both the number of nodes that are emulated and the package dropping percentage should be easy to change for testing purposes.
See [Appendix](#appendices) for a brief tutorial. **We expect a test coverage of at least 50\%** [^2].

**M5**: *Containerization*. **[5p]**.
You must be able to spin up a network of nodes on a single machine. **The network must consist of at least 50 nodes, each in its own container**[^3].
You may spin up and take down the network in any way you like, but you will likely save a lot of time if you either use a script or an orchestration solution[^4] to start and stop the network.

**M6**: *Lab report*. **[5p]**.
You must continuously work on and update a lab report. The required contents of the report are outlined later in the Method section.

**M7**: *Concurrency and thread safety*. **[6p]**.
To complete this objective, you must use some form of concurrency construct, such as threads, to make the handling of messages concurrent.
You must also be able to account for how you guarantee thread safety, via locks or otherwise.

### Qualifying
**U1**: *Object expiration*. **[2p]**.
Each node associates each data object it stores with a certain *Time-To-Live* (TTL).
When the TTL expires, the data object in question is silently deleted.
However, every time the data object is requested and transmitted, the TTL is to be reset.
You may decide the TTL yourself.
It should, however, be changeable so that you can demonstrate that the object expiration mechanism you build does work as intended.

**U2**: *Object expiration delay*. **[2p]**.
To prevent files from expiring, the node that originally uploaded each object sends a `refresh` command to the nodes having copies of it to prevent them from being deleted.
In particular, the command resets the TTL for the refreshed data object without actually requesting it.
As long as the uploading node can contact the storing nodes, the object in question should never expire.
Completing this objective requires that U1 is also completed.

**U3**: *Forget CLI command*. **[2p]**.
To allow the original uploader of an object to stop refreshing it, you add the `forget` CLI command, which accepts the hash of the object that is no longer to be refreshed.
Completing this objective requires that U2 is also completed.

**U4**: *RESTful application interface*. **[2p]**.
A CLI interface may be useful for humans with terminals, but it makes it difficult to integrate your storage network into web applications or other applications.
To remedy this, you will make every node also provide a RESTful as described at: [https://www.ics.uci.edu/~fielding/pubs/dissertation/rest\_arch\_style.htm](https://www.ics.uci.edu/~fielding/pubs/dissertation/rest\_arch\_style.htm).
HTTP interface with the following endpoints:
1. `POST /objects`:
Each message sent to the endpoint must contain a data object in its HTTP body.
If the operation is successful, the node must reply with `201 CREATED` containing both the contents of the uploaded object and a `Location: /objects/{hash}` header, where `{hash}` must be substituted for the hash of the uploaded object.
2. `GET /objects/{hash}`:
The `{hash}` portion of the HTTP path is to be substituted for the hash of the object.
A successful call should result in the contents of the object being responded to.

**U5**: *Higher unit test coverage*. **[2p]**.
This objective is considered completed if you can get **a unit test coverage of 80\% or higher**.

**U6**: *Implement the full/general Kademlia routing tree structure*. **[3p]**.
The branching parameter `b` (see subsection 4.2 of the Kademlia paper) must be freely adjustable.
You must have test cases that clearly demonstrate the correct behavior of the tree (in particular that buckets are split correctly for a given value of `b`).

## Further requirements
1. You are not allowed to use any packages that abstract too much functionality. An example of a package that hides too much functionality is any RPC package. If you are unsure, you can ask a TA, but make sure that you can explain what the library does and why it would be OK. If you want an example of a package which is at an OK abstraction level, you can refer to [protobuf package](https://github.com/golang/protobuf).
2. You must use UDP as the underlying communication protocol, not TCP. This also means that you cannot use any package that communicates through TCP.
3. You must implement a 160-bit RPC ID.
4. If not stated otherwise, you should adhere to the Kademlia description as closely as possible.

## Delimitations
To make this task manageable within the time frame of this course, we make these delimitations:
1. Data objects can **not** be explicitly deleted or modified. In other words, they must be *immutable*. They can still, however, expire or be lost.
2. Data objects can only be requested by their hashes. They have no other names or identifiers.
3. Data objects are always UTF-8 strings, which makes it possible to write them into terminals. You are allowed to set an arbitrary string length limit, such as 255 bytes.
4. Data objects are not saved to disk, which means that they disappear if you terminate the nodes that hold copies of them.
5. No communication is encrypted.
6. All network nodes have access to all stored data objects without needing any permissions.
7. The simplified, flat routing table is sufficient. Implementing the full/general tree is optional as a qualifying objective for extra points.

You are allowed to ignore any of these delimitations if you like, but be aware that it may complicate your implementation significantly.
You are not guaranteed a higher grade on the lab for making your implementation more complete, but you may consider the learning experience worth the effort.

## Method
You are expected to work loosely according to the principles of ASD, as we already mentioned in the Introduction.
This means that you need to break down and prioritise the objectives you want to complete and organise them into so-called *sprints*.
As each sprint will end with you presenting your progress to the teaching assistants, you may not decide the length of each sprint yourselves.
Make sure to look in Canvas for when each sprint ends.

## The Lab Report
While working, you are required to record your plans, progress and design decisions in a lab report, which you will present to the teaching assistants at the end of each sprint.
It is not expected to be completed until the last sprint review.
The report must contain the following:

1. The group number and the names of the two or three members of your group.
2. A link to your code repository, which could reside on e.g. [GitHub](https://github.com/).
3. A list of the frameworks and other tools you are using.
4. A system architecture description that also contains an implementation overview.
5. A description of your system's limitations and what possibilities exist for improvements.
6. A separate section for each sprint that contains (a) a sprint plan, (b) a backlog and (c) other reflections.

## Assessments
During each sprint, you will need to sign up for a sprint review at the end of the sprint.
Information about signing up will be announced via Canvas.
You are expected to be able to demonstrate the following at the sprint reviews:

**Sprint 0**: 
1. A working understanding of the Kademlia algorithm. The members of your group will be selected at random to answer some questions we prepared.
2. You must be able to spin up a network of at least 50 containerised nodes via a script or some orchestration software, as well as showing that any network member can send a message, of any kind, to any other. The nodes do not have to carry any Kademlia-related software at this point.
3. A plan for how you will organise your work in the coming sprint.
4. A lab report, which you submitted before the deadline announced via Canvas.
5. The report must at least account for your plans for sprint 1.

**Sprint 1**:
1. Objectives M1 to M7, if you plan on a higher grade. Otherwise, you need to be able to demonstrate how far you have come in completing these objectives.
2. What you did during the sprint, and a plan for how you will organise your work in the coming sprint.
3. A lab report, which you submitted before the deadline announced via Canvas.
4. The report must at least account for your plans and progress.
5. We expect you to have implemented a decent amount of the unit tests for sprint 1.

**Sprint 2**:
1. Objectives M1 to M7 and as many of the qualifying objectives as you like.
2. What did you do during the sprint?
3. A lab report, which you submitted before the deadline announced via Canvas. The report must be complete at this point.

Note that your performance on sprint reviews 0 and 1 has no bearing on your final grade on the lab.
The expectations are set to help you divide your work evenly across the duration of the course.

# Appendices
## Unit Testing
The file `routingtable_test.go` contains an incomplete unit test of the routing table. The test code ought to give you a rough idea of how to use the routing table. The function NewRoutingTable creates a new routing table and takes k (the replication factor) and the contact information of the local peer as arguments. Each peer in the network is represented by a Contact object, which contains a KademliaID, an IP address and a port number. The KademliaID is an opaque 160-bit integer, which could be generated using a random number generator or a hash of a UUID, for example. For testing purposes (as in the example below), it could also be hard-coded to some specific values.

```go
rt := NewRoutingTable(
    NewContact(NewKademliaID("FFFFFFFF00000000000000000000000000000000"), "localhost:8000")
)
rt.AddContact(NewContact(NewKademliaID("FFFFFFFF00000000000000000000000000000000"), "localhost:8001"))
rt.AddContact(NewContact(NewKademliaID("1111111100000000000000000000000000000000"), "localhost:8002"))
rt.AddContact(NewContact(NewKademliaID("1111111200000000000000000000000000000000"), "localhost:8002"))
rt.AddContact(NewContact(NewKademliaID("1111111300000000000000000000000000000000"), "localhost:8002"))
rt.AddContact(NewContact(NewKademliaID("1111111400000000000000000000000000000000"), "localhost:8002"))
rt.AddContact(NewContact(NewKademliaID("2111111400000000000000000000000000000000"), "localhost:8002"))
contacts := rt.FindClosestContacts(NewKademliaID("2111111400000000000000000000000000000000"), 20)
for i := range contacts {
    fmt.Println(contacts[i].String())
}
```

The example above creates a routing table and populates it with some fake contacts. The kademlia.FindClosestFunction returns other peers in the network that are close to the specified target. If you read the Kademlia paper, you will understand that distance is the XOR value of two identifiers. Play around with the code and make sure you fully understand every line of it.

## Some tips
Here's some advice we like to give students during sprint 0.
But it's probably better to put it in one place so everyone gets the same information and so that you can refer back to it later.

### Additional resources

The Kademlia paper leaves out some details. This page fills in some of those details:
[https://xlattice.sourceforge.net/components/protocol/kademlia/specs.html](https://xlattice.sourceforge.net/components/protocol/kademlia/specs.html)


### Testing
Achieving a minimum test coverage of 50 % is specified as a separate mandatory requirement (M4).
However, we strongly suggest that you don't treat it as a separate task to work on.
That is, **don't implement first and then add tests at the end!**

First of all, achieving the mandatory coverage will be much more difficult if you add tests after finishing the implementation.
Second, your tests should help you get the implementation working in the first place!
So, look at the tests as a tool to help you rather than a box to tick.
(In fact, if you write tests in parallel with the implementation, then the optional (non-mandatory) goal of 80 % (U5) should not be very difficult to achieve.)


### Concurrency and thread safety
Make sure you really understand race conditions, so that you know what problem it is you are trying to solve.
Think carefully about which parts of the code are run by more than one thread (or goroutine) and, in particular, which variables/data structures are accessed by different threads.

You can use old-fashioned locks to control access to critical regions.
That's a perfectly valid solution and perhaps one you are familiar with and find natural to think about.
However, there are other options, especially in Go, which has *channels* built into the language.
For example, one option is to let only a single goroutine have access to a particular data structure, and then other goroutines communicate with it using channels.
If you get used to this way of thinking, you may find this solution to be **simpler** than locks!

Either way, you can run your tests with the `-race` flag (see documentation: [Data Race Detector](https://go.dev/doc/articles/race_detector)).
This will instrument your code so that race conditions can be detected automatically.
**However**, how helpful this is depends on how good your tests are.
If there are race conditions that your tests never touch, then they will not be detected.
The race dector is designed to never have false positives.
(Running with `-race` is a dynamic rather than static analysis!)

### Report
The instructions say that the report needs to include "a system architecture description that also contains an implementation overview".
A common question is what this should look like.
Ultimately, the point is that you are supposed to communicate to someone how your system is designed.
What are the main components, and how do they communicate with each other, etc?
Just like you have to make choices about what the best way is to design and implement your solution, you have to make choices about what the best way is to communicate to someone else what you have done.

Try this: imagine that we change the assignment so that the students next year will get your implementation as a starting point and are asked to improve it, such as adding features.
*What information would they need so they quickly understand your implementation and can start modifying it?*
*What design choices should they be aware of?*

# Notes
[^1]: While the [official paper](https://pdos.csail.mit.edu/~petar/papers/maymounkov-kademlia-lncs.pdf) should be the document you first examine to understand the algorithm, there are plenty of complementary resources that can be helpful for clarifying some parts of that paper, such as [this interactive description](https://kelseyc18.github.io/kademlia\_vis/basics) or [this specification](http://xlattice.sourceforge.net/components/protocol/kademlia/specs.html).
[^2]: You can read more about test coverage in Go at [https://blog.golang.org/cover/](https://blog.golang.org/cover/). If you use an IDE like [GoLand](https://www.jetbrains.com/go/) which you can get a student license via your LTU e-mail address, tools for test coverage are built in.
[^3]: If you choose to use Docker as a container solution, which we recommend, you may want to use the Docker image described at [https://hub.docker.com/r/larjim/kademlialab/](https://hub.docker.com/r/larjim/kademlialab/) as a starting point.
[^4]: Docker Compose, which you can read about at [https://docs.docker.com/compose/](https://docs.docker.com/compose/), may be a good alternative if you choose to use Docker. There is already a Docker compose file named `docker-compose-lab.yml` in this repo.
