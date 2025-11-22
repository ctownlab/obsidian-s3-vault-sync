
*Created: 08-24-2025*
## Content
PARSEC's state is stored across multiple shards.[[]]

An agent calls `begin()` in it's broker to start a transaction. The broker then requests a ticket from the ticket machine / get a transaction ID. The agent uses `trylock` to to take out read/write locks on the appropriate shards. Upon a request a shard can grant the lock, wait for.another transaction to complete, or do a pre-empt. Once all reads/writes are completed, the broker calls `prepare()` and `commit()` once it gets a response from all the shards. The shard will then release locks and apply writes.  

**Transaction Lifecycle**
![[parsec_transaction_lifecycle.png]]

## Insights
* The result must be calculated in memory and then what's in memory gets translated to state during `commit()`.
## Further Study
1. Specifics about read and write locks
## Related Notes
1. [[PARSEC pre-empt Transaction Keys]]
## Tags
#parsec
## Citations
[1]

J. Lovejoy, A. Brownworth, M. Virza, and N. Narula, “PARSEC: Executing Smart Contracts in Parallel”.