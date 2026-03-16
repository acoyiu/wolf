import { onBeforeUnmount, onMounted, ref } from 'vue'

/**
 * @typedef {Object} SocketEnvelope
 * @property {string} type
 * @property {Object} payload
 */

/**
 * Connection helper with capped reconnect retries.
 * @param {() => string} urlFactory
 */
const HEARTBEAT_INTERVAL = 12000
const HEARTBEAT_TIMEOUT = 5000

export function useSocket(urlFactory) {
  const socket = ref(null)
  const status = ref('disconnected')
  const reconnectAttempts = ref(0)
  const lastMessage = ref(null)
  const errorMessage = ref('')

  let manualClose = false
  let heartbeatTimer = null
  let heartbeatTimeout = null

  function clearHeartbeat() {
    if (heartbeatTimer) { clearInterval(heartbeatTimer); heartbeatTimer = null }
    if (heartbeatTimeout) { clearTimeout(heartbeatTimeout); heartbeatTimeout = null }
  }

  function startHeartbeat(ws) {
    clearHeartbeat()
    heartbeatTimer = setInterval(() => {
      if (ws.readyState !== WebSocket.OPEN) return
      ws.send(JSON.stringify({ type: 'ping', payload: {} }))
      heartbeatTimeout = setTimeout(() => {
        ws.close()
      }, HEARTBEAT_TIMEOUT)
    }, HEARTBEAT_INTERVAL)
  }

  const connect = () => {
    if (socket.value && socket.value.readyState === WebSocket.OPEN) {
      return
    }

    const ws = new WebSocket(urlFactory())
    socket.value = ws
    status.value = reconnectAttempts.value > 0 ? 'reconnecting' : 'connecting'

    ws.onopen = () => {
      status.value = 'connected'
      reconnectAttempts.value = 0
      errorMessage.value = ''
      startHeartbeat(ws)
    }

    ws.onmessage = (event) => {
      if (heartbeatTimeout) { clearTimeout(heartbeatTimeout); heartbeatTimeout = null }
      try {
        const parsed = JSON.parse(event.data)
        lastMessage.value = parsed
      } catch {
        errorMessage.value = 'invalid_server_message'
      }
    }

    ws.onerror = () => {
      errorMessage.value = 'socket_error'
    }

    ws.onclose = () => {
      clearHeartbeat()
      if (manualClose) {
        status.value = 'disconnected'
        return
      }
      if (reconnectAttempts.value < 5) {
        reconnectAttempts.value += 1
        status.value = 'reconnecting'
        const delay = Math.min(reconnectAttempts.value * 1000, 5000)
        window.setTimeout(connect, delay)
      } else {
        status.value = 'failed'
        errorMessage.value = 'reconnect_failed'
      }
    }
  }

  /**
   * @param {string} type
   * @param {Object} payload
   */
  const send = (type, payload = {}) => {
    if (!socket.value || socket.value.readyState !== WebSocket.OPEN) {
      errorMessage.value = 'socket_not_connected'
      return false
    }
    socket.value.send(JSON.stringify({ type, payload }))
    return true
  }

  const close = () => {
    manualClose = true
    clearHeartbeat()
    if (socket.value) {
      socket.value.close()
    }
  }

  onMounted(connect)
  onBeforeUnmount(close)

  return {
    status,
    reconnectAttempts,
    errorMessage,
    lastMessage,
    connect,
    send,
    close,
  }
}
