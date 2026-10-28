const test = require('node:test')
const assert = require('node:assert/strict')
const fs = require('node:fs')
const vm = require('node:vm')
const source = fs.readFileSync('static/assets/js/summaryColors.js', 'utf8')
const context = { Chart: { defaults: {} } }
vm.createContext(context)
vm.runInContext(source, context)
const evaluate = expression => vm.runInContext(expression, context)

test('OS aliases share one display category while machines remain distinct', () => {
  const os = evaluate(`mergeChartItems([{key:'Mac',total:100},{key:'Macos',total:3},{key:'Linux',total:2}], 'os')`)
  assert.deepEqual(JSON.parse(JSON.stringify(os)), [{key:'Mac',total:103},{key:'Linux',total:2}])
  const machines = evaluate(`mergeChartItems([{key:'Macintosh.local',total:100},{key:'MacBook.local',total:3}], 'machine')`)
  assert.equal(machines.length, 2)
})

test('category case variants merge for display without changing the original data', () => {
  const items = [{key:'AI Coding',total:10},{key:'ai coding',total:4},{key:'Writing Docs',total:2},{key:'writing docs',total:1}]
  context.items = items
  const merged = evaluate(`mergeChartItems(items, 'category')`)
  assert.deepEqual(JSON.parse(JSON.stringify(merged)), [{key:'AI Coding',total:14},{key:'Writing Docs',total:3}])
  assert.equal(items[0].total, 10)
})

test('editor and category charts use distinct restrained categorical colors', () => {
  const editor = evaluate(`chartColor('editor', 'Claude code', 0)`)
  const category = evaluate(`chartColor('category', 'AI Coding', 0)`)
  assert.match(editor, /^#[0-9a-f]{6}$/i)
  assert.match(category, /^#[0-9a-f]{6}$/i)
  assert.notEqual(editor, category)
  assert.equal(editor, evaluate(`chartColor('editor', 'CLAUDE CODE', 5)`))
})
