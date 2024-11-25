## Intro
A blockchain system capable of sharding blocks, supporting centralized consensus and shard-based verification. It offers various configurations and collects data such as TPS and CPU usage. It provides four consensus algorithms: PBFT, HotStuff, Jolteon, and Symphony.

-----------------------
## How to use

Set the node storage location and other key parameters in the main.go, build the main program in the home directory and the main program in the node directory. Process the main program and add the parameter --type local to run multiple nodes locally for testing.
### Local test
We provide a local testing method and have packaged a Windows version of the program package. You can test it by modifying the following local.toml:

1. local.toml and witCon program, node program need to be in the same directory.
2. The directory where the node program is located, enter it into the nodePath of local.toml.
3. the local IP entered into local.toml.
4. Create a directory to run the node, and enter the directory into the path of local.toml.
5. If you have generated a non-windows node program, enter the name of the node program into the nodeName of local.toml.
6. Confirm the number of nodes you want to run, enter it into local.toml's nodes, note that this is a local run, it will generate the corresponding number of nodes' directories in path, with 0, 1, 2... respectively. To generate the node directories.
7. Type . \witCon.exe -type local to run.
8. The log will show the logs printed by all the nodes.

### Remote Test
**witCon** also supports a remote mode, allowing you to construct a network topology using a launcher and multiple reactors.

1. Start a launcher, listening on a specific port, and set the target number of nodes.
2. Start multiple reactors, targeting the launcher and connecting to the specified port.
3. Once the launcher connects to enough nodes, it automatically assigns identities to all nodes and customizes configuration files for each node based on the local configuration file.
4. Then, you can use the launcher's console commands to control the reactor nodes.

### witCon Start Command
- `./witcon -type launcher -count 4 -IP 192.168.1.52`  
  Starts the launcher, listening on IP `192.168.1.52` and preparing to connect with 4 consensus nodes (validation nodes will be determined based on the configuration).

- `./witcon -type reactor -IP 192.168.1.52`  
  Starts a reactor, automatically dialing to `192.168.1.52`. If there is no `sk` file in the node's directory, it will automatically generate one to represent its Account.

- `./witcon -type verify -IP 192.168.1.52`  
  Starts a verify node, automatically dialing to `192.168.1.52`. Similarly, if there is no `sk` file in the directory, one will be generated to represent its Account.

### witCon Console Commands
- `update`  
  Updates the configuration of all nodes.
- `start`  
  Starts the blockchain node program on all nodes.
- `stop`  
  Stops the blockchain node program on all nodes.
- `app`  
  Updates the node program on all nodes (the update package should be placed in the witCon program's directory).
- `collect`  
  Collects information such as TPS and CPU usage recorded by all nodes.

### Notes
- Except for the `stop` command, all other commands must be executed only after the `stop` command is issued, and all nodes have confirmed that the blockchain node program has stopped.


Other configuration meanings are as follows:
```editorconfig
logLvl #Log level, 3 is statistics only, 4 contains some debug logs.
consensus #consensus, symphony 0 , pbft 1, hotstuff 2, jolteon 3
txAmount #number of transactions per block
txSize #Size of every other transaction
viewChangeDuration #Timeout time
rtt #network latency
prePack #Whether to enable pipelined packaging  
SignatureVerify #Whether to skip signature verification  
signVerifyCore #Number of CPU cores allocated for signature verification (represented as the number of parallel threads)  
shardCount #Number of shards  
shardVerifyCore #Number of CPU cores for shard verification  
VerifyCount #Number of shard verification peers; if 0, it means there are no shard verification peers 
```

```Hardcoded Configuration  
common.value EthTxPath #Ethereum data source, usually the "dataset" directory, separated by block height 
``` 
-----------------------
## code structure

### chaincode
It can be developed through chaincode to implement smart contract functionality.

### cloud
Cloud controllers, which rapidly complete node deployment through mirroring, control node start/stop and configuration updates through a single controller.

### consensus
Includes implementations of pbft, hotstuff, jolteon, Symphony

### console
Allows the user to manipulate the node through the console after startup

### core
Blockchain core code, including transaction pools, state of the world, node permission control, etc.

### kyber-master
bls implementation for validating the effect of bls.

### node 
Node, start the node's main program.

### p2p
Simple TCP connection.

