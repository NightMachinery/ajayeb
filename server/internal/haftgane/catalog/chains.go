package catalog

// The 24 chain symbols, numbered in Age order: by the id of the one design
// each symbol frees, so the ten Age II targets come first. A symbol is named
// after that design; registration checks the providers.
const (
	ChainArcheryRange Chain = iota + 1
	ChainStables
	ChainAqueduct
	ChainCourthouse
	ChainStatue
	ChainCaravansery
	ChainForum
	ChainDispensary
	ChainLaboratory
	ChainLibrary
	ChainCircus
	ChainFortifications
	ChainSiegeWorkshop
	ChainGardens
	ChainPantheon
	ChainSenate
	ChainArena
	ChainHaven
	ChainLighthouse
	ChainAcademy
	ChainLodge
	ChainObservatory
	ChainStudy
	ChainUniversity
)
