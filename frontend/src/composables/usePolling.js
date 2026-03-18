import { ref, onBeforeUnmount } from 'vue'

/**
 * Polling composable for room state.
 * Used for phases that don't require real-time WebSocket updates.
 */
export function usePolling() {
  const isPolling = ref(false)
  const lastPollData = ref(null)
  const pollError = ref('')

  let pollTimer = null
  let currentRoomCode = ''
  let currentPlayerId = ''
  let currentInterval = 2000

  async function fetchState() {
    if (!currentRoomCode || !currentPlayerId) return null
    try {
      const res = await fetch(`/api/room/${currentRoomCode}/state?playerId=${currentPlayerId}`)
      if (!res.ok) {
        pollError.value = res.status === 404 ? 'room_not_found' : 'poll_error'
        return null
      }
      const data = await res.json()
      pollError.value = ''
      lastPollData.value = data
      return data
    } catch (e) {
      pollError.value = 'network_error'
      return null
    }
  }

  function startPolling(roomCode, playerId, intervalMs = 2000) {
    stopPolling()
    currentRoomCode = roomCode
    currentPlayerId = playerId
    currentInterval = intervalMs
    isPolling.value = true

    fetchState()

    pollTimer = window.setInterval(fetchState, intervalMs)
  }

  function stopPolling() {
    if (pollTimer) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
    isPolling.value = false
  }

  function changeInterval(intervalMs) {
    if (pollTimer && intervalMs !== currentInterval) {
      currentInterval = intervalMs
      stopPolling()
      startPolling(currentRoomCode, currentPlayerId, intervalMs)
    }
  }

  onBeforeUnmount(stopPolling)

  return {
    isPolling,
    lastPollData,
    pollError,
    startPolling,
    stopPolling,
    changeInterval,
    fetchState,
  }
}
