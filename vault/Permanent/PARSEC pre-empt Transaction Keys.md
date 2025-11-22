
*Created: 08-28-2025*
## Content
If a transaction attempts to lock an already locked key, and it has a lower ticket number than the current transaction, and the transaction is not in `prepare()` it will pre-empt it. The higher ticket transaction will then call `rollback()` and have to try again. 

## Related Notes
1. 
## Further Study

## Tags
#parsec
## Citations
[1]

J. Lovejoy, A. Brownworth, M. Virza, and N. Narula, “PARSEC: Executing Smart Contracts in Parallel”.