import assert from 'node:assert/strict'
import test from 'node:test'
import { getLineEdges, getVisibleLineEdges } from '../src/components/metroEdges.ts'
import type { MetroLine, Station } from '../src/types/metro.ts'

const line: MetroLine = {
  id: 1,
  name: 'Line',
  color: '#123abc',
  stations: [1, 2, 3, 4],
}

function visibleStations(ids: number[]): Map<number, Station> {
  return new Map(ids.map(id => [id, { id, name: `Station ${id}`, x: id, y: id }]))
}

test('explicit edges preserve a fork rather than chaining the station list', () => {
  const fork: MetroLine = { ...line, edges: [[1, 2], [2, 3], [2, 4]] }
  const edges = getVisibleLineEdges(fork, visibleStations([1, 2, 3, 4]))
  assert.deepEqual(edges.map(([from, to]) => [from.id, to.id]), [[1, 2], [2, 3], [2, 4]])
})

test('ring closing edge is preserved even with unique station IDs', () => {
  const ring: MetroLine = { ...line, stations: [1, 2, 3], edges: [[1, 2], [2, 3], [3, 1]] }
  const edges = getVisibleLineEdges(ring, visibleStations([1, 2, 3]))
  assert.deepEqual(edges.map(([from, to]) => [from.id, to.id]), [[1, 2], [2, 3], [3, 1]])
})

test('empty explicit edges do not fall back to station order', () => {
  assert.deepEqual(getLineEdges({ ...line, edges: [] }), [])
  assert.deepEqual(getVisibleLineEdges({ ...line, edges: [] }, visibleStations([1, 2, 3, 4])), [])
})

test('legacy schemas with absent or null edges retain adjacent connections', () => {
  const expected = [[1, 2], [2, 3], [3, 4]]
  assert.deepEqual(getLineEdges(line), expected)
  assert.deepEqual(getLineEdges({ ...line, edges: null }), expected)
})

test('year filtering removes incident edges without bridging the hidden station', () => {
  const graph: MetroLine = { ...line, edges: [[1, 2], [2, 3], [3, 4]] }
  const edges = getVisibleLineEdges(graph, visibleStations([1, 3, 4]))
  assert.deepEqual(edges.map(([from, to]) => [from.id, to.id]), [[3, 4]])
})

test('legacy filtering also avoids inventing a bridge across a hidden station', () => {
  const edges = getVisibleLineEdges(line, visibleStations([1, 3, 4]))
  assert.deepEqual(edges.map(([from, to]) => [from.id, to.id]), [[3, 4]])
})

test('edges with an unknown endpoint are skipped', () => {
  const graph: MetroLine = { ...line, edges: [[1, 99], [1, 2]] }
  const edges = getVisibleLineEdges(graph, visibleStations([1, 2]))
  assert.deepEqual(edges.map(([from, to]) => [from.id, to.id]), [[1, 2]])
})
