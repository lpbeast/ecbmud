package combat

// This exists so that characters and mobs can refer to it, and then also both
// implement it, in order to interact with each other in combat

type Combatant interface {
	EnterCombat(target Combatant)
	DoAutoAttack() (string, string)
	ReceiveDamage(dmg int)
	GetName() string
	GetDefense() int
	GetHP() int
	ExitCombat()
}
