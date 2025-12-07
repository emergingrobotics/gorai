// Package action provides long-running action patterns for Gorai.
//
// Actions are like services but support feedback during execution
// and can be canceled. They follow a request-response model where:
//   - A client sends a goal to a server
//   - The server processes the goal and sends periodic feedback
//   - The server sends a final result when complete
//   - The client can cancel goals at any time
//
// Example usage:
//
//	// Server side
//	server, err := action.NewServer[*MyGoal, *MyFeedback, *MyResult](
//	    node, "my_action",
//	    func(ctx context.Context, handle *action.GoalHandle[*MyGoal, *MyFeedback, *MyResult]) {
//	        for i := 0; i < 10; i++ {
//	            if handle.IsCanceling() {
//	                handle.SetCanceled(&MyResult{})
//	                return
//	            }
//	            handle.SendFeedback(&MyFeedback{Progress: float32(i) / 10})
//	            time.Sleep(time.Second)
//	        }
//	        handle.SetSucceeded(&MyResult{Success: true})
//	    },
//	)
//
//	// Client side
//	client, err := action.NewClient[*MyGoal, *MyFeedback, *MyResult](node, "my_action")
//	handle, err := client.SendGoal(ctx, &MyGoal{Target: 100})
//	for fb := range handle.Feedback() {
//	    fmt.Printf("Progress: %.0f%%\n", fb.Progress * 100)
//	}
//	result, err := handle.Wait(ctx)
package action
