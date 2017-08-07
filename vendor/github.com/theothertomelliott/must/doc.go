/*
Package must provides helper functions for validating output in unit tests. The output "must" be what we expect.

Must does not provide assertions, but follows a similar syntax that you might be used to from unit testing in other languages. You provide a testing.T, objects to be tested and an error message, and the must functions will raise a testing error if the expectations on the objects are not met. This error will contain additional context (such as an object diff) to help you identify the nature of the test failure.

For example:

 result := must.BeEqual(t, expected, got, "expectation not met")

Will trigger an error in t if got and expected are not the same. The message "expectation not met" will be included in the error along with a diff of expected and got.

*/
package must
