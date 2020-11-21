package rewards

import "github.com/theQRL/zond/config"

func GetBlockReward() uint64 {

	// TODO: Update Block Reward
	return 5 * config.GetDevConfig().ShorPerQuanta
}

func GetAttestorReward() uint64 {
	return 2 * config.GetDevConfig().ShorPerQuanta
}