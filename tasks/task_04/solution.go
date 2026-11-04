package main

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	if len(nums) < 2 {
		return Stats{}
	}

	firstDiff := nums[1] - nums[0]

	stats := Stats{
		Count: len(nums) - 1,
		Sum:   firstDiff,
		Min:   firstDiff,
		Max:   firstDiff,
	}

	for i := 2; i < len(nums); i++ {
		diff := nums[i] - nums[i-1]

		stats.Sum += diff

		if diff < stats.Min {
			stats.Min = diff
		}

		if diff > stats.Max {
			stats.Max = diff
		}
	}

	return stats
}
