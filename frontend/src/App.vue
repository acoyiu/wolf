<template>
  <main class="shell">
    <section class="glass panel status-bar">
      <div>
        <p class="label">連線狀態</p>
        <p class="value" :class="`state-${status}`">{{ statusText }}</p>
      </div>
      <div>
        <p class="label">房間</p>
        <p class="value">{{ room.roomCode || '-' }}</p>
      </div>
      <div>
        <p class="label">你</p>
        <p class="value">{{ myNickname || '-' }}</p>
      </div>
    </section>

    <section class="panel hero" v-if="view === 'lobby'">
      <h1>Wolfword</h1>
      <p>面對面遊玩，每人一支手機，同步進行。</p>

      <label class="field">
        暱稱
        <input v-model.trim="myNickname" maxlength="16" placeholder="輸入你的名字" />
      </label>

      <div class="lobby-grid">
        <article class="glass card" v-if="!isInviteMode">
          <h2>建立房間</h2>
          <label class="field">
            玩家
            <input v-model.number="targetPlayers" type="number" min="4" max="10" />
          </label>
          <label class="field">
            難度
            <select v-model="difficulty">
              <option value="easy">簡單</option>
              <option value="medium">中等</option>
              <option value="hard">困難</option>
            </select>
          </label>
          <button class="btn primary" @click="createRoom">建立</button>
        </article>

        <article class="glass card">
          <h2>加入房間</h2>
          <label class="field" v-if="!isInviteMode">
            房間代碼
            <input v-model.trim="joinCode" maxlength="6" placeholder="AB3K" />
          </label>
          <label class="field" v-else>
            邀請房號
            <input :value="joinCode" readonly />
          </label>
          <button class="btn" @click="joinRoom">{{ isInviteMode ? '加入邀請房' : '加入' }}</button>
        </article>
      </div>
    </section>

    <section class="panel" v-if="view === 'waiting'">
      <div class="panel-head">
        <h2>等待室</h2>
        <p>{{ room.players.length }}/{{ room.targetPlayers }} 人</p>
      </div>

      <div class="waiting-layout">
        <article class="glass card">
          <p class="label">房間代碼</p>
          <p class="code">{{ room.roomCode }}</p>
          <p class="label">分享連結</p>
          <p class="mono wrap">{{ shareUrl }}</p>
          <img v-if="qrDataUrl" class="qr" :src="qrDataUrl" alt="房間 QR Code" />
        </article>

        <article class="glass card">
          <p class="label">玩家</p>
          <ul class="players">
            <li v-for="p in room.players" :key="p.id">
              <span>{{ p.nickname }}</span>
              <span class="chip" v-if="p.isHost">房主</span>
            </li>
          </ul>
        </article>
      </div>

      <div class="actions">
        <button class="btn primary" @click="startGame" :disabled="!canStart">開始遊戲</button>
        <button class="btn" @click="leaveRoom">離開</button>
      </div>
    </section>

    <section class="panel" v-if="view === 'night'">
      <h2>夜晚階段</h2>
      <p class="role-badge">
        <component v-if="roleIconMap[effectiveRole]" :is="roleIconMap[effectiveRole]" :size="24" />
        <span>{{ roleText }}</span>
      </p>

      <article class="glass card" v-if="night.step === 1 && isHost">
        <p>請選擇祕密咒語。</p>
        <p class="label" v-if="selectedWord">已選擇: {{ selectedWord }}。等待其他玩家...</p>
        <div class="pill-grid">
          <button class="btn pill" v-for="word in night.candidates" :key="word" @click="pickWord(word)" :disabled="Boolean(selectedWord)">{{ word }}</button>
        </div>
      </article>

      <article class="glass card" v-else-if="night.revealWord">
        <p>請記住這個咒語：</p>
        <p class="word">{{ night.revealWord }}</p>
        <button class="btn" @click="nightConfirm" :disabled="nightConfirmed">{{ nightConfirmed ? '已確認' : '下一步' }}</button>
        <p class="label" v-if="nightConfirmed">等待其他玩家...</p>
      </article>

      <article class="glass card" v-else>
        <p class="night-flavor">{{ nightFlavorText }}</p>
        <button class="btn" @click="nightConfirm" :disabled="nightConfirmed">{{ nightConfirmed ? '已確認' : '下一步' }}</button>
        <p class="label" v-if="nightConfirmed">等待其他玩家...</p>
      </article>
    </section>

    <section class="panel" v-if="view === 'day'">
      <div class="panel-head">
        <h2>白天階段</h2>
        <div class="timer-display" v-if="timer.remainingMs > 0">
          <span class="timer-icon">⏱</span>
          <span class="timer-value" :class="{ 'timer-urgent': timer.remainingMs <= 30000 }">{{ formatTimer(timer.remainingMs) }}</span>
        </div>
      </div>
      <p class="role-badge">
        <component v-if="roleIconMap[effectiveRole]" :is="roleIconMap[effectiveRole]" :size="20" />
        <span>{{ roleText }}</span>
      </p>

      <p class="label turn-hint" v-if="daySpeakerIdx >= 0">
        提問順序: <strong>{{ room.players[daySpeakerIdx]?.nickname }}</strong>
        <button class="btn-inline" @click="advanceSpeaker" v-if="isHost">下一位</button>
      </p>

      <article class="glass card" v-if="isHost">
        <p class="label">村長控制台</p>
        <div class="token-grid">
          <button class="btn token yes" @click="sendToken('yes')">是</button>
          <button class="btn token no" @click="sendToken('no')">否</button>
          <button class="btn token maybe" @click="sendToken('maybe')">或許</button>
          <button class="btn token close" @click="sendToken('close')">接近</button>
          <button class="btn token far" @click="sendToken('far')">差太多</button>
          <button class="btn token correct" @click="sendToken('correct')">正確</button>
        </div>
      </article>

      <article class="glass card">
        <p class="label">指示物摘要</p>
        <div class="token-summary">
          <span v-for="item in tokenSummaryItems" :key="item.key" class="chip token-chip" :class="`token ${item.className}`">{{ item.label }} ×{{ item.value }}</span>
        </div>
        <p class="label" style="margin-top:0.5rem">剩餘數量</p>
        <div class="token-grid token-dashboard">
          <div class="token-stat" :class="`token ${item.className}`" v-for="item in tokenStats" :key="item.key">
            <span>{{ item.label }}</span>
            <strong>{{ item.value }}</strong>
          </div>
        </div>
        <p class="label">歷史紀錄</p>
        <div class="history">
          <span class="chip token-chip" :class="`token ${tokenClass(token)}`" v-for="(token, idx) in day.history" :key="idx">{{ tokenLabel(token) }}</span>
          <span class="label" v-if="day.history.length === 0">目前還沒有回應。</span>
        </div>
      </article>
    </section>

    <section class="panel" v-if="view === 'vote'">
      <div class="panel-head">
        <h2>投票階段</h2>
        <div class="timer-display" v-if="timer.remainingMs > 0">
          <span class="timer-icon">⏱</span>
          <span class="timer-value" :class="{ 'timer-urgent': timer.remainingMs <= 15000 }">{{ formatTimer(timer.remainingMs) }}</span>
        </div>
      </div>
      <p class="role-badge">
        <component v-if="roleIconMap[effectiveRole]" :is="roleIconMap[effectiveRole]" :size="20" />
        <span>{{ roleText }}</span>
      </p>
      <p>{{ votePrompt }}</p>
      <p class="label vote-progress" v-if="voteProgress.total > 0">已投票: {{ voteProgress.voted }} / {{ voteProgress.total }}</p>
      <p class="label" v-if="!canVoteInCurrentMode">此回合僅狼人需要投票，請等待。</p>
      <div class="pill-grid" v-if="canVoteInCurrentMode">
        <button class="btn pill" v-for="p in voteCandidates" :key="p.id" @click="castVote(p.id)" :disabled="votedFor === p.id">
          {{ p.nickname }}
        </button>
      </div>
      <p v-if="votedFor && canVoteInCurrentMode" class="label">你已投給: {{ nameById(votedFor) }}</p>
    </section>

    <section class="panel" v-if="view === 'result'">
      <h2>遊戲結果</h2>
      <article class="glass card">
        <p class="winner">勝方: {{ winnerText }}</p>
        <p>原因: <span class="mono">{{ resultReasonText }}</span></p>
        <p>咒語: <strong>{{ result.word || '-' }}</strong></p>
      </article>
      <article class="glass card">
        <p class="label">角色列表</p>
        <ul class="players">
          <li v-for="p in room.players" :key="p.id">
            <span class="player-role-entry">
              <component v-if="roleIconMap[effectiveRoleOf(p.id)]" :is="roleIconMap[effectiveRoleOf(p.id)]" :size="28" />
              {{ p.nickname }}
            </span>
            <span class="mono">{{ roleByPlayer(p.id) }}</span>
          </li>
        </ul>
      </article>
      <article class="glass card" v-if="hasVoteData">
        <p class="label">投票明細</p>
        <ul class="players">
          <li v-for="p in room.players" :key="'vote-'+p.id">
            <span>{{ p.nickname }}</span>
            <span class="mono" v-if="result.votes[p.id]">→ {{ nameById(result.votes[p.id]) }}</span>
            <span class="mono muted-text" v-else>—</span>
          </li>
        </ul>
      </article>
      <div class="actions" style="justify-content:center">
        <button class="btn primary" @click="playAgain" v-if="isHost">再來一局</button>
        <p class="label" v-else style="text-align:center">等待房主開始下一局...</p>
      </div>
    </section>

    <transition name="toast">
      <aside class="toast" v-if="toastText">{{ toastText }}</aside>
    </transition>
  </main>
</template>

<script setup>
import { computed, reactive, ref, watch, onBeforeUnmount } from 'vue'
import QRCode from 'qrcode'
import { useSocket } from './composables/useSocket'
import RoleWerewolf from './icons/RoleWerewolf.vue'
import RoleSeer from './icons/RoleSeer.vue'
import RoleVillager from './icons/RoleVillager.vue'

const roleIconMap = { werewolf: RoleWerewolf, seer: RoleSeer, villager: RoleVillager }

const SESSION_KEY = 'wolfword.session'
const savedSession = loadSession()

const myNickname = ref(savedSession?.nickname || '')
const targetPlayers = ref(6)
const difficulty = ref('easy')
const inviteCodeFromUrl = (new URLSearchParams(window.location.search).get('roomCode') || '').toUpperCase()
const joinCode = ref(inviteCodeFromUrl || savedSession?.roomCode || '')
const toastText = ref('')
let resumeHintTimerId = 0
let pendingResumeNewId = ''

const playerId = ref('')
const myRole = ref('')
const mayorSecret = ref('')
const view = ref('lobby')
const shareUrl = ref('')
const qrDataUrl = ref('')
const votedFor = ref('')
const voteMode = ref('guess_wolf')
const nightConfirmed = ref(false)
const selectedWord = ref('')
const daySpeakerIdx = ref(-1)

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

const timer = reactive({
  phase: '',
  remainingMs: 0,
})
let timerIntervalId = 0
let timerTargetTime = 0

const voteProgress = reactive({
  voted: 0,
  total: 0,
})

const result = reactive({
  winner: '',
  reason: '',
  word: '',
  roles: {},
  mayorSecret: '',
  votes: {},
})

const wsUrl = () => {
  const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  return `${scheme}//${window.location.host}/ws`
}

const { status, reconnectAttempts, errorMessage, lastMessage, send } = useSocket(wsUrl)

const statusText = computed(() => {
  switch (status.value) {
    case 'connected': return '已連線'
    case 'connecting': return '連線中'
    case 'reconnecting': return '重新連線中'
    case 'failed': return '連線失敗'
    default: return '未連線'
  }
})

const isInviteMode = computed(() => Boolean(inviteCodeFromUrl))
const isHost = computed(() => room.players.some((p) => p.id === playerId.value && p.isHost))
const canStart = computed(() => isHost.value && room.players.length >= room.targetPlayers)
const roleText = computed(() => (myRole.value === 'mayor' ? `村長 (${roleName(mayorSecret.value || 'unknown')})` : roleName(myRole.value || 'unknown')))
const effectiveRole = computed(() => (myRole.value === 'mayor' ? (mayorSecret.value || '') : myRole.value))
const voteCandidates = computed(() => room.players.filter((p) => p.id !== playerId.value))
const votePrompt = computed(() => (voteMode.value === 'guess_seer' ? '狼人正在投票指認先知。' : '全體玩家投票找出狼人。'))
const canVoteInCurrentMode = computed(() => (voteMode.value !== 'guess_seer' ? true : effectiveRole.value === 'werewolf'))
const hasVoteData = computed(() => result.votes && Object.keys(result.votes).length > 0)
const winnerText = computed(() => {
  if (result.winner === 'villagers') return '村民陣營'
  if (result.winner === 'werewolves') return '狼人陣營'
  return '未定'
})
const resultReasonText = computed(() => formatReasonCode(result.reason || ''))

const nightFlavorText = computed(() => {
  if (night.step === 1) return '夜幕降臨…村長正在選擇祕密咒語，請耐心等待。'
  return '知情者正在確認咒語…請閉上眼睛，保持安靜。'
})

const tokenStats = computed(() => [
  { key: 'yes', label: '是', className: 'yes', value: day.remaining.yes },
  { key: 'no', label: '否', className: 'no', value: day.remaining.no },
  { key: 'maybe', label: '或許', className: 'maybe', value: day.remaining.maybe },
  { key: 'close', label: '接近', className: 'close', value: day.remaining.close },
  { key: 'far', label: '差太多', className: 'far', value: day.remaining.far },
  { key: 'correct', label: '正確', className: 'correct', value: day.remaining.correct },
])

const tokenSummaryItems = computed(() => {
  const used = {}
  for (const token of day.history) {
    used[token] = (used[token] || 0) + 1
  }
  const items = []
  const order = ['yes', 'no', 'maybe', 'close', 'far', 'correct']
  const labels = { yes: '是', no: '否', maybe: '或許', close: '接近', far: '差太多', correct: '正確' }
  for (const key of order) {
    if (used[key]) {
      items.push({ key, label: labels[key], className: key, value: used[key] })
    }
  }
  return items
})

onBeforeUnmount(() => {
  stopLocalTimer()
})

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

watch([playerId, myNickname, () => room.roomCode], () => {
  persistSession()
})

function handleMessage(msg) {
  const payload = msg.payload || {}

  if (msg.type !== 'error') {
    clearResumeHint()
  }

  switch (msg.type) {
    case 'connected':
      tryResumeSession(payload.playerId || '')
      break
    case 'session_resumed':
      pendingResumeNewId = ''
      if (payload.playerId) {
        playerId.value = payload.playerId
      }
      if (payload.roomCode) {
        room.roomCode = payload.roomCode
      }
      toast('session_resumed')
      break
    case 'room_created':
      hydrateRoom(payload)
      shareUrl.value = payload.joinUrl || ''
      view.value = 'waiting'
      break
    case 'room_state':
      hydrateRoom(payload)
      view.value = 'waiting'
      resetGameState()
      break
    case 'player_joined':
    case 'player_left':
      hydrateRoom(payload)
      if (view.value !== 'night' && view.value !== 'day' && view.value !== 'vote' && view.value !== 'result') {
        view.value = 'waiting'
      }
      break
    case 'player_reconnecting':
      toast(formatReconnecting(payload))
      break
    case 'player_reconnected':
      toast(formatReconnected(payload))
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
      nightConfirmed.value = false
      selectedWord.value = ''
      hapticFeedback('phase')
      break
    case 'night_reveal':
      view.value = 'night'
      night.step = payload.step || 2
      night.revealWord = payload.word || ''
      nightConfirmed.value = false
      selectedWord.value = ''
      hapticFeedback('reveal')
      break
    case 'phase_change':
      if (payload.phase === 'day') {
        view.value = 'day'
        day.history = []
        nightConfirmed.value = false
        selectedWord.value = ''
        initSpeakerOrder()
        hapticFeedback('phase')
      }
      break
    case 'timer_sync':
      syncTimer(payload)
      break
    case 'day_state':
      if (payload.remaining) {
        day.remaining = payload.remaining
      }
      if (Array.isArray(payload.history)) {
        day.history = payload.history
      }
      break
    case 'mayor_response':
      day.history.push(payload.token)
      if (payload.remaining) {
        day.remaining = payload.remaining
      }
      hapticFeedback('token')
      break
    case 'word_guessed':
      voteMode.value = 'guess_seer'
      voteProgress.voted = 0
      voteProgress.total = 0
      view.value = 'vote'
      hapticFeedback('phase')
      break
    case 'time_up':
    case 'tokens_depleted':
      voteMode.value = 'guess_wolf'
      voteProgress.voted = 0
      voteProgress.total = 0
      view.value = 'vote'
      hapticFeedback('phase')
      break
    case 'vote_state':
      voteMode.value = payload.voteType === 'guess_seer' ? 'guess_seer' : 'guess_wolf'
      votedFor.value = payload.votedFor || ''
      view.value = 'vote'
      break
    case 'vote_cast':
      if (typeof payload.votedCount === 'number') {
        voteProgress.voted = payload.votedCount
        voteProgress.total = payload.totalVoters || 0
      }
      hapticFeedback('token')
      break
    case 'vote_result':
      break
    case 'game_over':
      result.winner = payload.winner || ''
      result.reason = payload.reason || ''
      result.word = payload.word || ''
      result.roles = payload.roles || {}
      result.mayorSecret = payload.mayorSecret || ''
      result.votes = payload.votes || {}
      view.value = 'result'
      stopLocalTimer()
      clearSession()
      hapticFeedback('result')
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
      {
        const message = String(payload.message || 'error')
        if (message.startsWith('resume_')) {
          if (pendingResumeNewId) {
            playerId.value = pendingResumeNewId
            pendingResumeNewId = ''
          }
          const session = loadSession()
          const canFallbackJoin = Boolean(session?.roomCode && session?.nickname)
          clearSession()
          if (canFallbackJoin) {
            resetToLobby()
            myNickname.value = session.nickname
            joinCode.value = session.roomCode
            scheduleResumeHint()
            const ok = emit('join_room', {
              roomCode: session.roomCode,
              nickname: session.nickname,
            })
            if (!ok) {
              clearResumeHint()
              toast('reconnect_failed')
              resetToLobby()
            }
          } else {
            clearResumeHint()
            toast('reconnect_failed')
            resetToLobby()
          }
        } else {
          clearResumeHint()
          if (view.value === 'vote' && (message === 'already_voted' || message === 'cannot_vote_self' || message === 'invalid_target')) {
            votedFor.value = ''
          }
          toast(message)
        }
      }
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
  emit('create_room', {
    nickname: myNickname.value,
    targetPlayers: Math.min(10, Math.max(4, Number(targetPlayers.value) || 6)),
    difficulty: difficulty.value,
  })
}

function joinRoom() {
  if (!myNickname.value) return toast('nickname_required')
  if (!joinCode.value) return toast('room_code_required')
  emit('join_room', {
    roomCode: joinCode.value.toUpperCase(),
    nickname: myNickname.value,
  })
}

function leaveRoom() {
  emit('leave_room', {})
  resetToLobby()
}

function startGame() {
  emit('start_game', {})
}

function playAgain() {
  emit('play_again', {})
}

function pickWord(word) {
  if (selectedWord.value) {
    return
  }
  const ok = emit('night_pick_word', { word })
  if (ok) {
    selectedWord.value = word
  }
}

function nightConfirm() {
  if (nightConfirmed.value) {
    return
  }
  const ok = emit('night_confirm', {})
  if (ok) {
    nightConfirmed.value = true
  }
}

function sendToken(token) {
  emit('day_token', { token })
}

function tokenClass(token) {
  switch (token) {
    case 'yes':
    case 'no':
    case 'maybe':
    case 'close':
    case 'far':
    case 'correct':
      return token
    default:
      return ''
  }
}

function tokenLabel(token) {
  switch (token) {
    case 'yes': return '是'
    case 'no': return '否'
    case 'maybe': return '或許'
    case 'close': return '接近'
    case 'far': return '差太多'
    case 'correct': return '正確'
    default: return token
  }
}

function castVote(targetId) {
  if (!canVoteInCurrentMode.value) {
    toast('not_eligible_voter')
    return
  }
  const ok = emit('vote_cast', { target: targetId })
  if (ok) {
    votedFor.value = targetId
  }
}

function tryResumeSession(newPlayerId) {
  const session = loadSession()
  if (!session || !session.playerId || !session.roomCode || !session.nickname) {
    playerId.value = newPlayerId
    return
  }
  if (!myNickname.value) {
    myNickname.value = session.nickname
  }
  pendingResumeNewId = newPlayerId
  emit('resume_session', {
    playerId: session.playerId,
    roomCode: session.roomCode,
    nickname: session.nickname,
  })
}

function persistSession() {
  if (!playerId.value || !myNickname.value || !room.roomCode) {
    return
  }
  const session = {
    playerId: playerId.value,
    roomCode: room.roomCode,
    nickname: myNickname.value,
  }
  window.localStorage.setItem(SESSION_KEY, JSON.stringify(session))
}

function loadSession() {
  try {
    const raw = window.localStorage.getItem(SESSION_KEY)
    if (!raw) {
      return null
    }
    const parsed = JSON.parse(raw)
    return {
      playerId: String(parsed.playerId || ''),
      roomCode: String(parsed.roomCode || '').toUpperCase(),
      nickname: String(parsed.nickname || ''),
    }
  } catch {
    return null
  }
}

function clearSession() {
  window.localStorage.removeItem(SESSION_KEY)
}

function emit(type, payload) {
  const ok = send(type, payload)
  if (!ok) {
    toast('socket_not_connected')
  }
  return ok
}

function roleName(role) {
  switch (role) {
    case 'mayor': return '村長'
    case 'seer': return '先知'
    case 'werewolf': return '狼人'
    case 'villager': return '村民'
    default: return '未知'
  }
}

function roleByPlayer(id) {
  if (id === playerId.value && myRole.value === 'mayor') {
    return `村長 (${roleName(result.mayorSecret || mayorSecret.value || 'unknown')})`
  }
  return roleName(result.roles[id] || 'unknown')
}

function effectiveRoleOf(id) {
  const r = result.roles[id] || ''
  if (r === 'mayor') return result.mayorSecret || mayorSecret.value || ''
  return r
}

function nameById(id) {
  const found = room.players.find((p) => p.id === id)
  return found ? found.nickname : id
}

function formatReconnecting(payload) {
  const name = payload?.nickname || nameById(payload?.playerId || '')
  if (name && name !== payload?.playerId) {
    return `${name} 正在重新連線，請稍候。`
  }
  return '有玩家正在重新連線，請稍候。'
}

function formatReconnected(payload) {
  const name = payload?.nickname || nameById(payload?.playerId || '')
  if (name && name !== payload?.playerId) {
    return `${name} 已重新連線。`
  }
  return '玩家已重新連線。'
}

function syncTimer(payload) {
  const ms = Number(payload.remainingMs) || 0
  timer.phase = payload.phase || ''
  timer.remainingMs = ms
  timerTargetTime = Date.now() + ms
  stopLocalTimer()
  if (ms > 0) {
    timerIntervalId = window.setInterval(() => {
      const remaining = Math.max(0, timerTargetTime - Date.now())
      timer.remainingMs = remaining
      if (remaining <= 0) {
        stopLocalTimer()
      }
    }, 250)
  }
}

function stopLocalTimer() {
  if (timerIntervalId) {
    window.clearInterval(timerIntervalId)
    timerIntervalId = 0
  }
}

function formatTimer(ms) {
  const totalSec = Math.ceil(ms / 1000)
  const min = Math.floor(totalSec / 60)
  const sec = totalSec % 60
  return `${min}:${String(sec).padStart(2, '0')}`
}

function initSpeakerOrder() {
  if (room.players.length > 0) {
    const hostIdx = room.players.findIndex((p) => p.isHost)
    daySpeakerIdx.value = hostIdx >= 0 ? (hostIdx + 1) % room.players.length : 0
  } else {
    daySpeakerIdx.value = -1
  }
}

function advanceSpeaker() {
  if (room.players.length === 0) return
  daySpeakerIdx.value = (daySpeakerIdx.value + 1) % room.players.length
}

function hapticFeedback(type) {
  try {
    if (navigator.vibrate) {
      switch (type) {
        case 'phase':
          navigator.vibrate([100, 50, 100])
          break
        case 'reveal':
          navigator.vibrate([200])
          break
        case 'token':
          navigator.vibrate([50])
          break
        case 'result':
          navigator.vibrate([100, 80, 100, 80, 200])
          break
      }
    }
  } catch {
    // Vibration API not available
  }
}

function toast(message) {
  const text = formatToastMessage(message)
  toastText.value = text
  window.setTimeout(() => {
    if (toastText.value === text) {
      toastText.value = ''
    }
  }, 2500)
}

function scheduleResumeHint() {
  clearResumeHint()
  resumeHintTimerId = window.setTimeout(() => {
    resumeHintTimerId = 0
    toast('session_retry_join')
  }, 1000)
}

function clearResumeHint() {
  if (!resumeHintTimerId) {
    return
  }
  window.clearTimeout(resumeHintTimerId)
  resumeHintTimerId = 0
}

function formatToastMessage(message) {
  if (typeof message !== 'string') {
    return String(message)
  }
  if (message.startsWith('connection_lost_retry_')) {
    const attempt = message.replace('connection_lost_retry_', '')
    return `連線中斷，正在重試（第 ${attempt} 次）`
  }
  switch (message) {
    case 'invalid_message': return '收到無效訊息，請重新操作。'
    case 'unsupported_message_type': return '不支援的操作。'
    case 'invalid_payload': return '請求內容不完整，請重試。'
    case 'invalid_nickname': return '暱稱格式不正確。'
    case 'session_resumed': return '已恢復上一局連線。'
    case 'session_retry_join': return '正在恢復連線...'
    case 'nickname_required': return '請先輸入暱稱。'
    case 'room_code_required': return '請輸入房間代碼。'
    case 'socket_not_connected': return '尚未連線到伺服器。'
    case 'room_not_found': return '找不到房間，請確認代碼。'
    case 'room_full': return '房間已滿。'
    case 'nickname_already_taken': return '此暱稱已被使用，請換一個。'
    case 'player_not_found': return '找不到玩家資料，請重新加入。'
    case 'host_only': return '只有房主可以執行這個操作。'
    case 'not_enough_players': return '玩家人數不足，無法開始。'
    case 'game_already_started': return '遊戲已開始。'
    case 'game_not_found': return '目前沒有進行中的遊戲。'
    case 'word_library_unavailable': return '詞庫暫時不可用，請稍後再試。'
    case 'mayor_only': return '只有村長可以執行這個操作。'
    case 'invalid_phase': return '目前階段無法執行此操作。'
    case 'invalid_word': return '選擇的咒語無效。'
    case 'mayor_must_pick_word': return '請先由村長選擇咒語。'
    case 'token_exhausted': return '這個指示物已用完。'
    case 'invalid_token': return '無效的指示物。'
    case 'not_eligible_voter': return '你不是此回合可投票的玩家。'
    case 'already_voted': return '你已經投過票了。'
    case 'cannot_vote_self': return '不能投給自己。'
    case 'invalid_target': return '投票目標無效。'
    case 'resume_room_not_found': return '原房間不存在，請重新加入。'
    case 'resume_player_not_found': return '原連線不存在，正在重新加入。'
    case 'resume_not_available': return '目前無法恢復連線，請重新加入。'
    case 'resume_in_use': return '此連線已在其他裝置使用。'
    case 'host_disconnected': return '房主已離線，房間已關閉。'
    case 'room_closed': return '房間已關閉。'
    case 'game_ended': return '本局已結束。'
    case 'player_disconnected': return '有玩家斷線，遊戲已中止。'
    case 'reconnect_failed': return '重新連線失敗，已回到大廳。'
    default:
      if (/^[a-z0-9_]+$/i.test(message)) {
        return '操作失敗，請稍後再試。'
      }
      return message
  }
}

function formatReasonCode(reason) {
  switch (reason) {
    case 'word_guessed_seer_found': return '猜中咒語後，狼人成功找出先知。'
    case 'word_guessed_seer_safe': return '猜中咒語後，狼人未找出先知。'
    case 'word_missed_wolf_caught': return '未猜中咒語，但村民成功抓到狼人。'
    case 'word_missed_wolf_safe': return '未猜中咒語，狼人成功躲過投票。'
    case 'player_disconnected': return '有玩家斷線未能及時重連。'
    case 'host_disconnected': return '房主離線，房間已關閉。'
    case 'game_ended': return '本局已結束。'
    default: return reason || '-'
  }
}

function resetGameState() {
  votedFor.value = ''
  voteMode.value = 'guess_wolf'
  myRole.value = ''
  mayorSecret.value = ''
  night.step = 1
  night.candidates = []
  night.revealWord = ''
  nightConfirmed.value = false
  selectedWord.value = ''
  day.history = []
  day.remaining = { yes: 48, no: 48, maybe: 1, close: 1, far: 1, correct: 1 }
  result.winner = ''
  result.reason = ''
  result.word = ''
  result.roles = {}
  result.mayorSecret = ''
  result.votes = {}
  voteProgress.voted = 0
  voteProgress.total = 0
  daySpeakerIdx.value = -1
  stopLocalTimer()
  timer.phase = ''
  timer.remainingMs = 0
}

function resetToLobby() {
  clearResumeHint()
  clearSession()
  view.value = 'lobby'
  room.roomCode = ''
  room.targetPlayers = 0
  room.players = []
  shareUrl.value = ''
  qrDataUrl.value = ''
  resetGameState()
}
</script>
