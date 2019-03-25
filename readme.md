Run the bash script with the fillowing args. *Flags are optional

-c | *Flag to tell if a server should imediately crash after it has been elected leader and sent out its first heartbeat

-n=x | x = the number of nodes

-p=x | p = initial port the first node will take up. 
        Note, The servers will take up initial port p to p + number of nodes

-stdio | *Flag to tell whether the ouptu should all display to stdio or to numbered files. 


example:

5 nodes that will crash when elected leader. Their output will be in numbered files.
./publish.sh -c -n=5 -p=8000 

5 nodes that will not crash purposfully. Their output will be to the command line
./publish.sh -n=5 -p=8000 -stdio
