
*Created: 08-23-2025*
## Content
In PARSEC, the broker exposes an interface that the agent can use to convert high level requests into database transactions. The same broker can be used for multiple Virtual Machines, i.e. EVM/LUA. 

Broker's can reach the appropriate shard for a specific key using a directory service. To lock a key, a broker must obtain a ticket from a ticketing machine, which is used to uniquely identify and prioritize a transaction. Keys are locked and updated using 2-Phase commit.

![[parsec_agent_archetecture.png]]
## Insights
* 
## Further Study
* 
## Related Notes
1. [[PARSEC trylock]]
2. [[PARSEC Distributed Transactions]]
## Tags
#parsec
## Citations
[1]

J. Lovejoy, A. Brownworth, M. Virza, and N. Narula, “PARSEC: Executing Smart Contracts in Parallel”.