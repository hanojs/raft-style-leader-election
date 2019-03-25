## Raft-Style Leader Election 

This is a leader-election saubprotocol loosely based on [this](https://raft.github.io/raft.pdf) raft paper by "Diego Ongaro and John Ousterhout at Stanford University" 

This implementation relys on at least 51% of servers to be alive; however, the frequency of hung votes will drastically increase as the number of failed nodes reaches 50%. (No cleanup for failed nodes) 

### Publish.sh arguments...

-c | Flag to tell if a server should imediately crash after it has been elected leader and sent out its first heartbeat

-n=int | int = the number of nodes

-p=int | int = initial port the first node will take up. 
        Note, The servers will take up initial port p to p + number of nodes

-stdio | Flag to tell whether the output should display to stdio or to numbered files. 


### Examples:

5 nodes that will crash when elected leader. Their output will be in numbered files.

```bash 
./publish.sh -c -n=5 -p=8000 
```

5 nodes that will not crash purposfully. Their output will be to the command line.

```bash
./publish.sh -n=5 -p=8000 -stdio
```
