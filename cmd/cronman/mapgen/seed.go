// Package mapgen provides an API for interacting with Rust map generation
// related mechanisms such as seeds, salts, etc.
package mapgen

import (
	"math/rand"

	"github.com/tjper/rustcron/cmd/cronman/model"
)

// GenerateSeed produces a seed that may be used to generate a map.
func GenerateSeed(size model.MapSizeKind) uint32 {
	var seed uint32
	switch size {
	case model.MapSizeSmall:
		seed = smallSeeds[rand.Intn(len(smallSeeds))]
	case model.MapSizeMedium:
		seed = mediumSeeds[rand.Intn(len(mediumSeeds))]
	case model.MapSizeLarge:
		seed = largeSeeds[rand.Intn(len(largeSeeds))]
	case model.MapSizeXLarge:
		seed = xLargeSeeds[rand.Intn(len(xLargeSeeds))]
	}

	return seed
}

// GenerateSalt produces a salt that may be used to generate a salt.
func GenerateSalt() uint32 {
	const max = 1000000
	return uint32(rand.Intn(max) + 1)
}

// smallSeeds are a slice of seeds that may be used to generate small Rust
// maps. These seeds have been hand selected to ensure a certain level of map
// quality.
var smallSeeds = []uint32{
	6033273,
	1459613893,
	88203632,
	1755999879,
	1278812382,
	1205429122,
	746525786,
	1517732640,
	1687200486,
	561539878,
	944132297,
	1272461318,
	139824936,
	1554996233,
	1823786156,
	1227356052,
	12093,
	506564568,
	242703041,
	732955033,
	1698061217,
	1181902159,
	2026159974,
	1196125075,
	1100316547,
	12468733,
	397495240,
	532754503,
	866793198,
	1122961061,
	2136468472,
	1440700782,
	132160969,
	1626826965,
	906398336,
	1587726743,
	159694894,
	1732553294,
	1020401210,
	1746762492,
	910507915,
}

// mediumSeeds are a slice of seeds that may be used to generate medium Rust
// maps. These seeds have been hand selected to ensure a certain level of map
// quality.
var mediumSeeds = []uint32{
	1452398625,
	1721279200,
	1310747615,
	26118,
	609601883,
	1193580610,
	1666421876,
	1966038332,
	1077198869,
	1103527533,
	49916095,
	2057401206,
	475238776,
	366343815,
	1791248682,
	6990642,
	1952951533,
	1502551439,
	35008921,
	1434202518,
	380484144,
	625749893,
	49342014,
	1791734938,
	1900448871,
	1486219016,
	91898457,
	610684414,
	194131515,
	222014563,
	1589958522,
	67496873,
	33,
	875165,
	64535612,
	4209999,
	13402336,
	349293237,
	13606970,
	42132132,
	926712091,
}

// largeSeeds are a slice of seeds that may be used to generate large Rust
// maps. These seeds have been hand selected to ensure a certain level of map
// quality.
var largeSeeds = []uint32{
	818172190,
	837190762,
	2811903,
	319877794,
	284690465,
	871985164,
	1742665867,
	455896655,
	478089395,
	690403,
	1172864681,
	411319480,
	988164013,
	1464417384,
	1549611900,
	1105334554,
	664732122,
	688157781,
	459057476,
	284723962,
	1636898093,
	492257539,
	656474672,
	2093292736,
	2089586306,
	1553300726,
	180790196,
	577787725,
	1383250143,
	1910077965,
	1123934608,
	835303425,
	1710689041,
	1336369169,
	39244688,
	613337887,
	1962954118,
	1075169197,
	916139288,
	1425725945,
	968941738,
}

// xLargeSeeds are a slice of seeds that may be used to generate x-large Rust
// maps. These seeds have been hand selected to ensure a certain level of map
// quality.
var xLargeSeeds = []uint32{
	443346119,
	160974,
	886029803,
	1983233550,
	1794420626,
	1357671590,
	16227857,
	1513463613,
	700171300,
	22102,
	833359885,
	905801828,
	811045939,
	1926668725,
	577865162,
	2543,
	1239396442,
	1924460680,
	1582631611,
	112604315,
	433230231,
	1523521952,
	975632708,
	1137459187,
	979517110,
	388582260,
	1636286521,
	2106372894,
	699640892,
	1499701537,
	348762346,
	1628447491,
	1846476232,
	1408273073,
	730705568,
	1517295485,
	2113342044,
	1077360243,
	687784319,
	1107740085,
	179576666,
}
