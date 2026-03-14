import { test, expect } from '@playwright/test'

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
  const roleEl = page.locator('.role-display span')
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
  await expect(player.page.getByText('請點擊下一步，保持全員同步。')).toBeVisible()
  await expect(player.page.getByText('請選擇祕密咒語。')).toHaveCount(0)
}

async function expectNightStep2View(player, chosenWord) {
  if (!player.roleInfo.isMayor && (player.roleInfo.effectiveRole === 'seer' || player.roleInfo.effectiveRole === 'werewolf')) {
    await expect(player.page.getByText('請記住這個咒語：')).toBeVisible({ timeout: 10000 })
    await expect(player.page.locator('.word')).toHaveText(chosenWord)
    return
  }

  await expect(player.page.getByText('請點擊下一步，保持全員同步。')).toBeVisible({ timeout: 10000 })
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

test('4 players can finish one round from lobby to result', async ({ browser }) => {
  const players = []

  try {
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

    await host.page.getByRole('button', { name: '開始遊戲' }).click()
    await Promise.all(
      players.map((p) => expect(p.page.getByRole('heading', { name: '夜晚階段' })).toBeVisible({ timeout: 10000 })),
    )

    for (const player of players) {
      player.roleInfo = await waitAndParseRole(player.page)
    }

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
      players.map((p) => expect(p.page.getByRole('heading', { name: '遊戲結果' })).toBeVisible({ timeout: 10000 })),
    )
  } finally {
    await Promise.all(players.map((p) => p.context.close()))
  }
})
