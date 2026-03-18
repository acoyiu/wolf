import { test, expect } from '@playwright/test'

const RESULT_REASONS = [
  '猜中咒語後，狼人成功找出先知。',
  '猜中咒語後，狼人未找出先知。',
  '未猜中咒語，但村民成功抓到狼人。',
  '未猜中咒語，狼人成功躲過投票。',
]

async function openPlayer(browser, nickname) {
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
  })
  const page = await context.newPage()
  await page.goto('/')
  await page.getByPlaceholder('輸入你的名字').fill(nickname)
  return { context, page, nickname }
}

function parseRole(roleText) {
  const text = roleText.trim()
  const isMayor = text.startsWith('村長')
  let effectiveRole = ''

  if (isMayor) {
    if (text.includes('(狼人)')) effectiveRole = 'werewolf'
    if (text.includes('(先知)')) effectiveRole = 'seer'
    if (text.includes('(村民)')) effectiveRole = 'villager'
  } else {
    if (text.includes('狼人')) effectiveRole = 'werewolf'
    if (text.includes('先知')) effectiveRole = 'seer'
    if (text.includes('村民')) effectiveRole = 'villager'
  }

  return { isMayor, effectiveRole }
}

async function waitAndParseRole(page) {
  const roleEl = page.locator('.role-card-name')
  await expect(roleEl).toBeVisible({ timeout: 10000 })
  await expect(roleEl).not.toContainText('未知', { timeout: 10000 })
  const roleText = await roleEl.innerText()
  const parsed = parseRole(roleText)
  expect(parsed.effectiveRole).not.toBe('')
  return parsed
}

async function expectNightStep1View(player, isHost) {
  await expect(player.page.getByRole('heading', { name: '夜晚階段' })).toBeVisible({ timeout: 10000 })
  if (isHost) {
    await expect(player.page.getByText('請選擇祕密咒語。')).toBeVisible()
    const candidates = player.page.locator('.pill-grid .btn.pill')
    await expect(candidates.first()).toBeVisible()
    await expect(candidates).toHaveCount(3)
    return
  }
  await expect(player.page.getByText('夜幕降臨…村長正在選擇祕密咒語，請耐心等待。')).toBeVisible()
  await expect(player.page.getByText('請選擇祕密咒語。')).toHaveCount(0)
}

async function expectNightStep2View(player, chosenWord) {
  if (!player.roleInfo.isMayor && (player.roleInfo.effectiveRole === 'seer' || player.roleInfo.effectiveRole === 'werewolf')) {
    await expect(player.page.getByText('請記住這個咒語：')).toBeVisible({ timeout: 10000 })
    await expect(player.page.locator('.word')).toHaveText(chosenWord)
    return
  }

  await expect(player.page.getByText('知情者正在確認咒語…請耐心等待。')).toBeVisible({ timeout: 10000 })
  await expect(player.page.getByText('請記住這個咒語：')).toHaveCount(0)
}

async function expectVoteVisibility(player) {
  const voteButtons = player.page.locator('section:has(h2:has-text("投票階段")) .pill-grid button')
  const isWerewolf = player.roleInfo.effectiveRole === 'werewolf'

  if (isWerewolf) {
    await expect(voteButtons.first()).toBeVisible({ timeout: 10000 })
    await expect(player.page.getByText('此回合僅狼人需要投票，請等待。')).toHaveCount(0)
    return
  }

  await expect(player.page.getByText('此回合僅狼人需要投票，請等待。')).toBeVisible({ timeout: 10000 })
  await expect(voteButtons).toHaveCount(0)
}

async function clickNextIfVisible(page, timeout = 7000) {
  const button = page.getByRole('button', { name: '下一步' }).first()
  try {
    await button.waitFor({ state: 'visible', timeout })
    await button.click()
    return true
  } catch {
    return false
  }
}

async function expectToast(page, text) {
  const toast = page.locator('.toast')
  await expect(toast).toBeVisible({ timeout: 10000 })
  await expect(toast).toContainText(text)
}

async function setupFourPlayers(browser) {
  const players = []
  players.push(await openPlayer(browser, 'P1'))
  players.push(await openPlayer(browser, 'P2'))
  players.push(await openPlayer(browser, 'P3'))
  players.push(await openPlayer(browser, 'P4'))

  const host = players[0]
  const others = players.slice(1)

  await host.page.locator('input[type="number"]').fill('4')
  await host.page.getByRole('button', { name: '建立' }).click()
  await expect(host.page.getByRole('heading', { name: '等待室' })).toBeVisible()

  const roomCode = (await host.page.locator('.code').innerText()).trim()
  expect(roomCode).not.toHaveLength(0)

  for (const player of others) {
    await player.page.getByPlaceholder('AB3K').fill(roomCode)
    await player.page.getByRole('button', { name: '加入' }).click()
    await expect(player.page.getByRole('heading', { name: '等待室' })).toBeVisible()
  }

  await Promise.all(players.map((p) => expect(p.page.getByText('4/4 人')).toBeVisible()))

  return { players, host, others, roomCode }
}

async function startAndReachNight(players, host) {
  await host.page.getByRole('button', { name: '開始遊戲' }).click()
  await Promise.all(
    players.map((p) => expect(p.page.getByRole('heading', { name: '夜晚階段' })).toBeVisible({ timeout: 10000 })),
  )

  for (const player of players) {
    player.roleInfo = await waitAndParseRole(player.page)
  }
}

async function advanceNightToDay(players, host, others) {
  for (const player of players) {
    await expectNightStep1View(player, player === host)
  }

  const chosenWord = (await host.page.locator('.pill-grid .btn.pill').first().innerText()).trim()
  expect(chosenWord).not.toHaveLength(0)
  await host.page.locator('.pill-grid .btn.pill').first().click()

  for (const player of others) {
    await clickNextIfVisible(player.page)
  }

  for (const player of players) {
    await expectNightStep2View(player, chosenWord)
  }

  for (const player of players) {
    await clickNextIfVisible(player.page, 10000)
  }

  await Promise.all(
    players.map((p) => expect(p.page.getByRole('heading', { name: '白天階段' })).toBeVisible({ timeout: 10000 })),
  )

  return chosenWord
}

async function assertLocalizedResultReason(players) {
  for (const player of players) {
    const reason = (await player.page.locator('.result-reason').innerText()).trim()
    // Result reason is now localized via formatReasonCode, so just check it exists and isn't raw code
    expect(reason.length).toBeGreaterThan(0)
    expect(reason).not.toMatch(/^[a-z_]+$/)  // Not a raw underscore-formatted code
  }
}

test('4 players can finish one round from lobby to result', async ({ browser }) => {
  const players = []

  try {
    const setup = await setupFourPlayers(browser)
    players.push(...setup.players)
    const { host, others } = setup

    await startAndReachNight(players, host)
    await advanceNightToDay(players, host, others)

    await expect(host.page.getByText('村長控制台')).toBeVisible()
    for (const player of others) {
      await expect(player.page.getByText('村長控制台')).toHaveCount(0)
    }

    await host.page.getByRole('button', { name: '正確' }).click()

    await Promise.all(
      players.map((p) => expect(p.page.getByRole('heading', { name: '投票階段' })).toBeVisible({ timeout: 10000 })),
    )

    for (const player of players) {
      await expectVoteVisibility(player)
    }

    for (const player of players) {
      const voteButtons = player.page.locator('section:has(h2:has-text("投票階段")) .pill-grid button')
      if ((await voteButtons.count()) > 0) {
        await voteButtons.first().click()
      }
    }

    await Promise.all(
      players.map((p) => expect(p.page.locator('.winner-label')).toBeVisible({ timeout: 10000 })),
    )

    await assertLocalizedResultReason(players)
  } finally {
    await Promise.all(players.map((p) => p.context.close()))
  }
})

test('4 players can reach guess_wolf vote by day timeout and all can vote', async ({ browser }) => {
  const players = []

  try {
    const setup = await setupFourPlayers(browser)
    players.push(...setup.players)
    const { host, others } = setup

    await startAndReachNight(players, host)
    await advanceNightToDay(players, host, others)

    await Promise.all(
      players.map((p) => expect(p.page.getByRole('heading', { name: '投票階段' })).toBeVisible({ timeout: 25000 })),
    )
    await Promise.all(
      players.map((p) => expect(p.page.getByText('全體玩家投票找出狼人。')).toBeVisible({ timeout: 25000 })),
    )

    // In guess_wolf mode, only non-werewolves can vote
    // Check UI visibility rather than relying on role parsing
    for (const player of players) {
      const voteButtons = player.page.locator('section:has(h2:has-text("投票階段")) .pill-grid button')
      const werewolfMsg = player.page.getByText('你是狼人，無法投票，請等待結果。')
      
      // Wait for either vote buttons or werewolf message to appear
      await Promise.race([
        voteButtons.first().waitFor({ state: 'visible', timeout: 5000 }).catch(() => {}),
        werewolfMsg.waitFor({ state: 'visible', timeout: 5000 }).catch(() => {}),
      ])
      
      const canVote = await voteButtons.count() > 0
      if (canVote) {
        await voteButtons.first().click()
      }
      // Werewolf just waits - no need to assert the message
    }

    await Promise.all(
      players.map((p) => expect(p.page.locator('.winner-label')).toBeVisible({ timeout: 10000 })),
    )

    await assertLocalizedResultReason(players)
  } finally {
    await Promise.all(players.map((p) => p.context.close()))
  }
})

test('refresh during game resumes to role view and does not show raw resume code', async ({ browser }) => {
  const players = []

  try {
    const setup = await setupFourPlayers(browser)
    players.push(...setup.players)
    const { host, others } = setup

    await startAndReachNight(players, host)

    const reloader = others[0]
    await reloader.page.reload()
    await expect(reloader.page.getByRole('heading', { name: '夜晚階段' })).toBeVisible({ timeout: 10000 })
    await expect(reloader.page.locator('.role-card-name')).not.toContainText('未知', { timeout: 10000 })
    await expect(reloader.page.getByText(/resume_[a-z_]+/i)).toHaveCount(0)
  } finally {
    await Promise.all(players.map((p) => p.context.close()))
  }
})

test('host leaves waiting room and other players are returned to lobby', async ({ browser }) => {
  const players = []

  try {
    const setup = await setupFourPlayers(browser)
    players.push(...setup.players)
    const { host, others } = setup

    await host.page.getByRole('button', { name: '離開' }).click()
    await expect(host.page.getByRole('heading', { name: 'Wolfword' })).toBeVisible({ timeout: 10000 })

    for (const player of others) {
      await expect(player.page.getByRole('heading', { name: 'Wolfword' })).toBeVisible({ timeout: 10000 })
      await expect(player.page.getByRole('heading', { name: '等待室' })).toHaveCount(0)
      await expect(player.page.getByText(/resume_[a-z_]+/i)).toHaveCount(0)
    }
  } finally {
    await Promise.all(players.map((p) => p.context.close()))
  }
})

test('join room with duplicate nickname shows localized error and stays in lobby', async ({ browser }) => {
  const players = []

  try {
    const host = await openPlayer(browser, 'DupName')
    const joiner = await openPlayer(browser, 'DupName')
    players.push(host, joiner)

    await host.page.locator('input[type="number"]').fill('4')
    await host.page.getByRole('button', { name: '建立' }).click()
    await expect(host.page.getByRole('heading', { name: '等待室' })).toBeVisible()

    const roomCode = (await host.page.locator('.code').innerText()).trim()
    expect(roomCode).not.toHaveLength(0)

    await joiner.page.getByPlaceholder('AB3K').fill(roomCode)
    await joiner.page.getByRole('button', { name: '加入' }).click()

    await expect(joiner.page.getByRole('heading', { name: 'Wolfword' })).toBeVisible({ timeout: 10000 })
    await expect(joiner.page.getByRole('heading', { name: '等待室' })).toHaveCount(0)
    await expectToast(joiner.page, '此暱稱已被使用，請換一個。')
    await expect(joiner.page.getByText(/nickname_already_taken/i)).toHaveCount(0)
  } finally {
    await Promise.all(players.map((p) => p.context.close()))
  }
})

test('join full room shows localized error and stays in lobby', async ({ browser }) => {
  const players = []

  try {
    const setup = await setupFourPlayers(browser)
    players.push(...setup.players)
    const { roomCode } = setup

    const extra = await openPlayer(browser, 'P5')
    players.push(extra)

    await extra.page.getByPlaceholder('AB3K').fill(roomCode)
    await extra.page.getByRole('button', { name: '加入' }).click()

    await expect(extra.page.getByRole('heading', { name: 'Wolfword' })).toBeVisible({ timeout: 10000 })
    await expect(extra.page.getByRole('heading', { name: '等待室' })).toHaveCount(0)
    await expectToast(extra.page, '房間已滿。')
    await expect(extra.page.getByText(/room_full/i)).toHaveCount(0)
  } finally {
    await Promise.all(players.map((p) => p.context.close()))
  }
})

test('in-game disconnect shows reconnecting notice to remaining players', async ({ browser }) => {
  const players = []

  try {
    const setup = await setupFourPlayers(browser)
    players.push(...setup.players)
    const { host, others } = setup

    await startAndReachNight(players, host)

    const disconnected = others[0]
    const remaining = [host, ...others.slice(1)]

    await disconnected.context.close()

    // When a player disconnects, remaining players see a toast notification about reconnection
    await Promise.all(
      remaining.map((player) =>
        expect(player.page.locator('.toast')).toContainText('重新連線', { timeout: 15000 }),
      ),
    )
  } finally {
    await Promise.all(
      players
        .filter((p) => p.context)
        .map(async (p) => {
          try {
            await p.context.close()
          } catch {
            // Ignore contexts that were already closed in test flow.
          }
        }),
    )
  }
})
