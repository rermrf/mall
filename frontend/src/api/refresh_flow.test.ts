import assert from 'node:assert/strict'

import { createRefreshFlow } from './refresh_flow.js'

async function testBuildRefreshHeaders() {
  const flow = createRefreshFlow()

  assert.deepEqual(flow.buildRefreshHeaders('refresh-token'), {
    'X-Refresh-Token': 'refresh-token',
  })
}

async function testRejectsPendingWaitersOnFailure() {
  const flow = createRefreshFlow()
  flow.begin()

  const waiter = flow.waitForToken()
  const expected = new Error('refresh failed')
  flow.fail(expected)

  await assert.rejects(waiter, expected)
  assert.equal(flow.isRefreshing(), false)
}

async function testResolvesPendingWaitersOnSuccess() {
  const flow = createRefreshFlow()
  flow.begin()

  const waiter = flow.waitForToken()
  flow.succeed('new-access-token')

  assert.equal(await waiter, 'new-access-token')
  assert.equal(flow.isRefreshing(), false)
}

async function main() {
  await testBuildRefreshHeaders()
  await testRejectsPendingWaitersOnFailure()
  await testResolvesPendingWaitersOnSuccess()
}

main().catch((err) => {
  console.error(err)
  process.exit(1)
})
