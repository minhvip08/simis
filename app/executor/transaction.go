package executor

type TransactionExecutor struct {
	blockingMgr *BlockingCommandManager
}

func NewTransactionExecutor(blockingMgr *BlockingCommandManager) *TransactionExecutor {
	return &TransactionExecutor{
		blockingMgr: blockingMgr,
	}
}
