# Scheduler improvements

* Clean up migration-related code in the scheduler

* Fix presentation

  * Actually establish sync between vkQueueSubmit we did and the vkQueuePresent.

* Bring back speculative signals

* Make the scheduler group things together somehow, to avoid unnecessary changes
  of state that could limit how much work can be in-flight on the device, etc.

* Shrink command buffer cache. This is rather low priority as we don't really
  grow additional command buffers in steady state.

# Memory management

* Implement the small allocator

* Shrink memory cache using goal-based heuristic. Drop the cache when we're
  about to reach the memory goal, and set goal to live size * 2.

  Also try shrinking memory using time heuristic. Assume that `vkAllocateMemory`
  of bigger size takes more time, and we want it to take no more than some fixed
  amount of program's time.

I hate allocators. Would be a lot nicer if we could just have the same address
space and memory on both host and device, and if necessary, things could be
moved to host or device memory by `madvise`-ing them.

# Shaders

* Change gpu.NewFunc to take optional function parameters so we can e.g. specify RT payload size etc that way

# P̸͇̙͝ͅͅṙ̷̲̪̊ẽ̸͓̺s̶̨̻̋̏̒́e̴͕̣͛͆n̵͉̹̖̪̎t̶̡̧͔̃̽a̵̠͆ͅt̸̘̼͉̻͝ï̸̭ȯ̵̭̯̖͒̂̋n̴̖͂͐͝͝

* ?̷̡̕?̸͔̫̗͊͂ͅ?̴̧̛̣͚̟̒͊
