
*Created: 08-23-2025*
## Content
Clients send raw Ethereum transactions to agents via `eth_sendRawTransaction`. For each transaction the agent sets up a host containing context (gas limit, timestamp, origin account, etc). Information about locked keys is kept in memory for use. 

To execute the agent will deserialize the transaction, checks that the nonce matches the current one on the transaction. It locks a key for the receipt, then creates an evm_message and calls `call` to execute it. The agent then waits for `call` to return and attaches the list of modified keys to the receipt. The agent then responds to the user who can retrieve their receipt via the `get_ethTransactionReceipt`. 

`call` implements core EVM semantics, including charging gas, deploying contracts, transferring funds, and evoking the EVM if needed. 

## Related Notes
1. 
## Further Study
1. More into the EVM primitives `eth_sendRawTransaction`, `get_ethTransactionReceipt`
2. Are `call` and the host interface part of the broker or something else?
## Tags
#parsec 
## Citations
[1]

J. Lovejoy, A. Brownworth, M. Virza, and N. Narula, “PARSEC: Executing Smart Contracts in Parallel”.