package common

//---------------- Articles of Association ---------------//

type IArticlesOfAssociation interface {
	GetContributionRule() IContributionRule
	GetWithdrawalRule() IWithdrawalRule
	GetAuditCost() IAuditCost
	GetPunishment() IPunishment

	SetContributionRule(contributionRule IContributionRule)
	SetWithdrawalRule(withdrawalRule IWithdrawalRule)
	SetAuditCost(auditCost IAuditCost)
	SetPunishment(punishment IPunishment)
}

type ArticlesOfAssociation struct {
	contributionRule IContributionRule
	withdrawalRule   IWithdrawalRule
	auditCost        IAuditCost
	punishment       IPunishment
}

func (a *ArticlesOfAssociation) GetContributionRule() IContributionRule {
	return a.contributionRule
}

func (a *ArticlesOfAssociation) GetWithdrawalRule() IWithdrawalRule {
	return a.withdrawalRule
}

func (a *ArticlesOfAssociation) GetAuditCost() IAuditCost {
	return a.auditCost
}

func (a *ArticlesOfAssociation) GetPunishment() IPunishment {
	return a.punishment
}

func (a *ArticlesOfAssociation) SetContributionRule(contributionRule IContributionRule) {
	a.contributionRule = contributionRule
}

func (a *ArticlesOfAssociation) SetWithdrawalRule(withdrawalRule IWithdrawalRule) {
	a.withdrawalRule = withdrawalRule
}

func (a *ArticlesOfAssociation) SetAuditCost(auditCost IAuditCost) {
	a.auditCost = auditCost
}

func (a *ArticlesOfAssociation) SetPunishment(punishment IPunishment) {
	a.punishment = punishment
}

func CreateArticlesOfAssociation(contributionRule IContributionRule, withdrawalRule IWithdrawalRule, auditCost IAuditCost, punishment IPunishment) *ArticlesOfAssociation {
	return &ArticlesOfAssociation{
		contributionRule: contributionRule,
		withdrawalRule:   withdrawalRule,
		auditCost:        auditCost,
		punishment:       punishment,
	}
}

//--------------- Contribution Strategies ---------------//

type IContributionRule interface {
	GetExpectedContributionAmount(agentScore int) int
	SetContributionAmount(amount int) // This can be removed or changed depending on future extensions
	// An extension could be to treat the contribution amount as a percentage of the agent score
}

type FixedContributionRule struct {
	contributionAmount int
}

func (f *FixedContributionRule) GetExpectedContributionAmount(agentScore int) int {
	// Agent score can be used if this were percentage based
	return f.contributionAmount
}

// Can be removed if we want to keep it fixed in future implementations
func (f *FixedContributionRule) SetContributionAmount(amount int) {
	f.contributionAmount = amount
}

func CreateFixedContributionRule(amount int) IContributionRule {
	return &FixedContributionRule{
		contributionAmount: amount,
	}
}

//--------------- Withdrawal Strategies ---------------//

type IWithdrawalRule interface {
	GetExpectedWithdrawalAmount(agentScore int) int
	SetWithdrawalAmount(amount int) // This can be removed or changed depending on future extensions
	// An extension could be to treat the withdrawal amount as a percentage of the agent score, could add the common pool to this as well maybe
}

type FixedWithdrawalRule struct {
	withdrawalAmount int
}

func (f *FixedWithdrawalRule) GetExpectedWithdrawalAmount(agentScore int) int {
	return f.withdrawalAmount
}

// Can be removed if we want to keep it fixed in future implementations
func (f *FixedWithdrawalRule) SetWithdrawalAmount(amount int) {
	f.withdrawalAmount = amount
}

func CreateFixedWithdrawalRule(amount int) IWithdrawalRule {
	return &FixedWithdrawalRule{
		withdrawalAmount: amount,
	}
}

//--------------- Audit Strategies ---------------//

type IAuditCost interface {
	GetAuditCost() int
	SetAuditCost(cost int) // This can be removed or changed depending on future extensions
}

type FixedAuditCost struct {
	auditCost int
}

func (f *FixedAuditCost) GetAuditCost() int {
	return f.auditCost
}

// Can be removed if we want to keep it fixed in future implementations
func (f *FixedAuditCost) SetAuditCost(cost int) {
	f.auditCost = cost
}

func CreateFixedAuditCost(cost int) IAuditCost {
	return &FixedAuditCost{
		auditCost: cost,
	}
}

//--------------- Punishment Strategies ---------------//

type IPunishment interface {
	GetPunishment() int
	SetPunishment(punishment int) // This can be removed or changed depending on future extensions
}

type FixedPunishment struct {
	punishment int
}

func (f *FixedPunishment) GetPunishment() int {
	return f.punishment
}

// Can be removed if we want to keep it fixed in future implementations
func (f *FixedPunishment) SetPunishment(punishment int) {
	f.punishment = punishment
}

func CreateFixedPunishment(punishment int) IPunishment {
	return &FixedPunishment{
		punishment: punishment,
	}
}
