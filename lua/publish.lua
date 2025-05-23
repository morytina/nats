wrk.method = "POST"
wrk.headers["Content-Type"] = "application/json"
wrk.body = [[
{
  "topicName": "sns-wrk-test",
  "message": "hello from wrk",
  "subject": "sns.wrk.test"
}
]]