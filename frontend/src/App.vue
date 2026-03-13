<template>
  <main class="shell">
    <section class="glass panel status-bar">
      <div>
        <p class="label">Connection</p>
        <p class="value" :class="`state-${status}`">{{ status }}</p>
      </div>
      <div>
        <p class="label">Room</p>
        <p class="value">{{ room.roomCode || '-' }}</p>
      </div>
      <div>
        <p class="label">You</p>
        <p class="value">{{ myNickname || '-' }}</p>
      </div>
    </section>

    <section class="panel hero" v-if="view === 'lobby'">
      <h1>Wolfword</h1>
      <p>Play face-to-face, use your own phones, and sync in real time.</p>

      <label class="field">
        Nickname
        <input v-model.trim="myNickname" maxlength="16" placeholder="Your name" />
      </label>

      <div class="lobby-grid">
        <article class="glass card" v-if="!isInviteMode">
          <h2>Create Room</h2>
          <label class="field">
            Players
            <input v-model.number="targetPlayers" type="number" min="4" max="10" />
          </label>
          <label class="field">
            Difficulty
            <select v-model="difficulty">
              <option value="easy">easy</option>
              <option value="medium">medium</option>
              <option value="hard">hard</option>
            </select>
          </label>
          <button class="btn primary" @click="createRoom">Create</button>
        </article>

        <article class="glass card">
          <h2>Join Room</h2>
          <label class="field" v-if="!isInviteMode">
            Room Code
            <input v-model.trim="joinCode" maxlength="6" placeholder="AB3K" />
          </label>
          <label class="field" v-else>
            Invite Room Code
            <input :value="joinCode" readonly />
          </label>
          <button class="btn" @click="joinRoom">{{ isInviteMode ? 'Join Invite Room' : 'Join' }}</button>
        </article>
      </div>
    </section>

    <section class="panel" v-if="view === 'waiting'">
      <div class="panel-head">
        <h2>Waiting Room</h2>
        <p>{{ room.players.length }}/{{ room.targetPlayers }} players</p>
      </div>

      <div class="waiting-layout">
        <article class="glass card">
          <p class="label">Room Code</p>
          <p class="code">{{ room.roomCode }}</p>
          <p class="label">Share URL</p>
          <p class="mono wrap">{{ shareUrl }}</p>
          <img v-if="qrDataUrl" class="qr" :src="qrDataUrl" alt="Room QR code" />
        </article>

        <article class="glass card">
          <p class="label">Players</p>
          <ul class="players">
            <li v-for="p in room.players" :key="p.id">
              <span>{{ p.nickname }}</span>
              <span class="chip" v-if="p.isHost">Host</span>
            </li>
          </ul>
        </article>
      </div>

      <div class="actions">
        <button class="btn primary" @click="startGame" :disabled="!canStart">Start Game</button>
        <button class="btn" @click="leaveRoom">Leave</button>
      </div>
    </section>

    <section class="panel" v-if="view === 'night'">
      <h2>Night Phase</h2>
      <p class="label">Role: {{ roleText }}</p>

      <article class="glass card" v-if="night.step === 1 && isHost">
        <p>Pick the secret word.</p>
        <div class="pill-grid">
          <button class="btn pill" v-for="word in night.candidates" :key="word" @click="pickWord(word)">{{ word }}</button>
        </div>
      </article>

      <article class="glass card" v-else-if="night.revealWord">
        <p>Memorize this word:</p>
        <p class="word">{{ night.revealWord }}</p>
        <button class="btn" @click="nightConfirm">Next</button>
      </article>

      <article class="glass card" v-else>
        <p>Tap next to keep all screens in sync.</p>
        <button class="btn" @click="nightConfirm">Next</button>
      </article>
    </section>

    <section class="panel" v-if="view === 'day'">
      <div class="panel-head">
        <h2>Day Phase</h2>
        <p>Ask verbally, answer with tokens.</p>
      </div>

      <article class="glass card" v-if="isHost">
        <p class="label">Mayor Controls</p>
        <div class="token-grid">
          <button class="btn token yes" @click="sendToken('yes')">YES</button>
          <button class="btn token no" @click="sendToken('no')">NO</button>
          <button class="btn token maybe" @click="sendToken('maybe')">MAYBE</button>
          <button class="btn token close" @click="sendToken('close')">CLOSE</button>
          <button class="btn token far" @click="sendToken('far')">FAR</button>
          <button class="btn token correct" @click="sendToken('correct')">CORRECT</button>
        </div>
      </article>

      <article class="glass card">
        <p class="label">Remaining</p>
        <p class="mono wrap">{{ tokenRemainingText }}</p>
        <p class="label">History</p>
        <div class="history">
          <span class="chip" v-for="(token, idx) in day.history" :key="idx">{{ token }}</span>
        </div>
      </article>
    </section>

    <section class="panel" v-if="view === 'vote'">
      <h2>Vote Phase</h2>
      <p>{{ votePrompt }}</p>
      <div class="pill-grid">
        <button class="btn pill" v-for="p in voteCandidates" :key="p.id" @click="castVote(p.id)" :disabled="votedFor === p.id">
          {{ p.nickname }}
        </button>
      </div>
      <p v-if="votedFor" class="label">You voted: {{ nameById(votedFor) }}</p>
    </section>

    <section class="panel" v-if="view === 'result'">
      <h2>Game Result</h2>
      <article class="glass card">
        <p class="winner">Winner: {{ result.winner || '-' }}</p>
        <p>Reason: <span class="mono">{{ result.reason || '-' }}</span></p>
        <p>Word: <strong>{{ result.word || '-' }}</strong></p>
      </article>
      <article class="glass card">
        <p class="label">Roles</p>
        <ul class="players">
          <li v-for="p in room.players" :key="p.id">
            <span>{{ p.nickname }}</span>
            <span class="mono">{{ roleByPlayer(p.id) }}</span>
          </li>
        </ul>
      </article>
    </section>

    <transition name="toast">
      <aside class="toast" v-if="toastText">{{ toastText }}</aside>
    </transition>
  </main>
</template>

<script setup>
import { computed, reactive, ref, watch } from 'vue'
import QRCode from 'qrcode'
import { useSocket } from './composables/useSocket'

const myNickname = ref('')
const targetPlayers = ref(6)
const difficulty = ref('easy')
const inviteCodeFromUrl = (new URLSearchParams(window.location.search).get('roomCode') || '').toUpperCase()
const joinCode = ref(inviteCodeFromUrl)
const toastText = ref('')

const playerId = ref('')
const myRole = ref('')
const mayorSecret = ref('')
const view = ref('lobby')
const shareUrl = ref('')
const qrDataUrl = ref('')
const votedFor = ref('')
const voteMode = ref('guess_wolf')

const room = reactive({
  roomCode: '',
  targetPlayers: 0,
  players: [],
})

const night = reactive({
  step: 1,
  candidates: [],
  revealWord: '',
})

const day = reactive({
  remaining: { yes: 48, no: 48, maybe: 1, close: 1, far: 1, correct: 1 },
  history: [],
})

const result = reactive({
  winner: '',
  reason: '',
  word: '',
  roles: {},
  mayorSecret: '',
})

const statusClass = {
  connected: 'ok',
  connecting: 'pending',
  reconnecting: 'pending',
  failed: 'bad',
  disconnected: 'bad',
}

const wsUrl = () => {
  const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${scheme}//${window.location.host}/ws`
}

const { status, reconnectAttempts, errorMessage, lastMessage, send } = useSocket(wsUrl)

const isInviteMode = computed(() => Boolean(inviteCodeFromUrl))
const isHost = computed(() => room.players.some((p) => p.id === playerId.value && p.isHost))
const canStart = computed(() => isHost.value && room.players.length >= room.targetPlayers)
const roleText = computed(() => (myRole.value === 'mayor' ? `mayor (${mayorSecret.value || 'unknown'})` : myRole.value || 'unknown'))
const voteCandidates = computed(() => room.players.filter((p) => p.id !== playerId.value))
const votePrompt = computed(() => (voteMode.value === 'guess_seer' ? 'Werewolves vote for the Seer.' : 'All players vote for a Werewolf.'))
const tokenRemainingText = computed(() => JSON.stringify(day.remaining))

watch(shareUrl, async (value) => {
  if (!value) {
    qrDataUrl.value = ''
    return
  }
  try {
    qrDataUrl.value = await QRCode.toDataURL(value, { width: 180, margin: 1 })
  } catch {
    qrDataUrl.value = ''
  }
})

watch(errorMessage, (message) => {
  if (!message) return
  toast(message)
  if (message === 'reconnect_failed') {
    resetToLobby()
  }
})

watch(reconnectAttempts, (attempt) => {
  if (attempt > 0) {
    toast(`connection_lost_retry_${attempt}`)
  }
})

watch(lastMessage, (msg) => {
  if (!msg || !msg.type) return
  handleMessage(msg)
})

function handleMessage(msg) {
  const payload = msg.payload || {}

  switch (msg.type) {
    case 'connected':
      playerId.value = payload.playerId || ''
      break
    case 'room_created':
      hydrateRoom(payload)
      shareUrl.value = payload.joinUrl || ''
      view.value = 'waiting'
      break
    case 'player_joined':
    case 'player_left':
      hydrateRoom(payload)
      view.value = 'waiting'
      break
    case 'role_assigned':
      myRole.value = payload.role || ''
      break
    case 'mayor_secret':
      mayorSecret.value = payload.secretRole || ''
      break
    case 'night_step':
      view.value = 'night'
      night.step = payload.step || 1
      night.candidates = payload.candidates || []
      night.revealWord = ''
      break
    case 'night_reveal':
      view.value = 'night'
      night.step = payload.step || 2
      night.revealWord = payload.word || ''
      break
    case 'phase_change':
      if (payload.phase === 'day') {
        view.value = 'day'
        day.history = []
      }
      break
    case 'mayor_response':
      day.history.push(payload.token)
      if (payload.remaining) {
        day.remaining = payload.remaining
      }
      break
    case 'word_guessed':
      voteMode.value = 'guess_seer'
      view.value = 'vote'
      break
    case 'time_up':
    case 'tokens_depleted':
      voteMode.value = 'guess_wolf'
      view.value = 'vote'
      break
    case 'vote_cast':
      break
    case 'vote_result':
      break
    case 'game_over':
      result.winner = payload.winner || ''
      result.reason = payload.reason || ''
      result.word = payload.word || ''
      result.roles = payload.roles || {}
      result.mayorSecret = payload.mayorSecret || ''
      view.value = 'result'
      break
    case 'game_aborted':
      toast(payload.reason || 'game_aborted')
      resetToLobby()
      break
    case 'room_closed':
      toast(payload.reason || 'room_closed')
      resetToLobby()
      break
    case 'error':
      toast(payload.message || 'error')
      break
    default:
      break
  }
}

function hydrateRoom(payload) {
  room.roomCode = payload.roomCode || room.roomCode
  room.targetPlayers = payload.targetPlayers || room.targetPlayers
  room.players = Array.isArray(payload.players) ? payload.players : room.players
}

function createRoom() {
  if (!myNickname.value) return toast('nickname_required')
  send('create_room', {
    nickname: myNickname.value,
    targetPlayers: Math.min(10, Math.max(4, Number(targetPlayers.value) || 6)),
    difficulty: difficulty.value,
  })
}

function joinRoom() {
  if (!myNickname.value) return toast('nickname_required')
  if (!joinCode.value) return toast('room_code_required')
  send('join_room', {
    roomCode: joinCode.value.toUpperCase(),
    nickname: myNickname.value,
  })
}

function leaveRoom() {
  send('leave_room', {})
  resetToLobby()
}

function startGame() {
  send('start_game', {})
}

function pickWord(word) {
  send('night_pick_word', { word })
}

function nightConfirm() {
  send('night_confirm', {})
}

function sendToken(token) {
  send('day_token', { token })
}

function castVote(targetId) {
  votedFor.value = targetId
  send('vote_cast', { target: targetId })
}

function roleByPlayer(id) {
  if (id === playerId.value && myRole.value === 'mayor') {
    return `mayor (${result.mayorSecret || mayorSecret.value || 'unknown'})`
  }
  return result.roles[id] || 'unknown'
}

function nameById(id) {
  const found = room.players.find((p) => p.id === id)
  return found ? found.nickname : id
}

function toast(message) {
  toastText.value = message
  window.setTimeout(() => {
    if (toastText.value === message) {
      toastText.value = ''
    }
  }, 2500)
}

function resetToLobby() {
  view.value = 'lobby'
  room.roomCode = ''
  room.targetPlayers = 0
  room.players = []
  shareUrl.value = ''
  qrDataUrl.value = ''
  votedFor.value = ''
  myRole.value = ''
  mayorSecret.value = ''
  night.step = 1
  night.candidates = []
  night.revealWord = ''
  day.history = []
  day.remaining = { yes: 48, no: 48, maybe: 1, close: 1, far: 1, correct: 1 }
  result.winner = ''
  result.reason = ''
  result.word = ''
  result.roles = {}
  result.mayorSecret = ''
}
</script>
