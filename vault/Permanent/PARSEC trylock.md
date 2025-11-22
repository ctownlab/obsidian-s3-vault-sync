
*Created: 08-23-2025*
## Content
`trylock` executes between the Agent and the Broker. The agent will first execute a `begin()`, which initializes a transaction. The agent then makes a series of `trylock` calls to the broker where each specifies a key on the shard. If keys depend on previous `trylock` results, the agent can use `trylock` to upgrade a previously held readlock to a write. The agent then responds with a commit message that can specify new values that it holds writelocks for. 

![[parsec_trylock_implementation.png]]
## Insights
* This seems to be a big part of the innovation of PARSEC but it is really simple. It is just locking and updating keys in a distributed state.
## Questions
* What about this makes it so innovative?
## Related Notes
1. [[PARSEC Broker]]
2. [[PARSEC Distributed Transactions]]
## Tags
#parsec
## Citations
[1]

J. Lovejoy, A. Brownworth, M. Virza, and N. Narula, “PARSEC: Executing Smart Contracts in Parallel”.