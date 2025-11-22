
*Created: 08-23-2025*
## Content
PARSEC (Parallel Architecture for Scalably Executing Smart Contracts) is a platform for executing smart contracts with linear scalability. It is centralized and does not require a global ordering. 

The architecture works between an agent and a distributed state. The agent runs a VM, i.e. EVM, and commits to state as if it has exclusive access. This is done through a primitive called `trylock` and an interface called a Broker. The database consists of simple key-value data structures. 
## Insights
* This is completely centralized
## Further Study
* This raises the question what exactly is a virtual machine? What is the commonality between a PARSEC virtual machine, a hypervisor virtual machine, and the java virtual machine?
## Related Notes
1. [[PARSEC trylock]]
2. [[PARSEC Broker]]
3. [[PARSEC Distributed Transactions]]
4. [[Parsec EVM Implementation]]
## Tags
#parsec
## Citations
[1]

J. Lovejoy, A. Brownworth, M. Virza, and N. Narula, “PARSEC: Executing Smart Contracts in Parallel”.
