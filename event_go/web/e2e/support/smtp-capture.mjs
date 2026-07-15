import net from 'node:net'
import http from 'node:http'

const messages = []

function commandReply(socket, state, line) {
  const command = line.toUpperCase()
  if (command.startsWith('EHLO')) {
    socket.write('250-localhost\r\n250-AUTH PLAIN\r\n250 OK\r\n')
  } else if (command.startsWith('HELO')) {
    socket.write('250 localhost\r\n')
  } else if (command.startsWith('AUTH PLAIN')) {
    socket.write('235 Authentication successful\r\n')
  } else if (command.startsWith('MAIL FROM:')) {
    state.from = line.slice('MAIL FROM:'.length).trim()
    socket.write('250 OK\r\n')
  } else if (command.startsWith('RCPT TO:')) {
    state.to = line.slice('RCPT TO:'.length).trim().replace(/^<|>$/g, '')
    socket.write('250 OK\r\n')
  } else if (command === 'DATA') {
    state.inData = true
    state.data = ''
    socket.write('354 End data with <CR><LF>.<CR><LF>\r\n')
  } else if (command === 'RSET') {
    state.from = ''
    state.to = ''
    socket.write('250 OK\r\n')
  } else if (command === 'QUIT') {
    socket.end('221 Bye\r\n')
  } else {
    socket.write('250 OK\r\n')
  }
}

const smtp = net.createServer((socket) => {
  const state = { buffer: '', data: '', from: '', to: '', inData: false }
  socket.write('220 localhost ESMTP Soulmark test capture\r\n')
  socket.on('data', (chunk) => {
    state.buffer += chunk.toString('utf8')
    while (state.buffer) {
      if (state.inData) {
        state.data += state.buffer
        state.buffer = ''
        const end = state.data.indexOf('\r\n.\r\n')
        if (end < 0) return
        const body = state.data.slice(0, end)
        state.buffer = state.data.slice(end + 5)
        messages.push({ to: state.to, from: state.from, body, received_at: new Date().toISOString() })
        state.inData = false
        state.data = ''
        socket.write('250 Message accepted\r\n')
        continue
      }
      const end = state.buffer.indexOf('\r\n')
      if (end < 0) return
      const line = state.buffer.slice(0, end)
      state.buffer = state.buffer.slice(end + 2)
      if (line) commandReply(socket, state, line)
    }
  })
})

const api = http.createServer((request, response) => {
  const url = new URL(request.url || '/', 'http://127.0.0.1')
  if (url.pathname === '/health') {
    response.writeHead(200, { 'Content-Type': 'application/json' }).end('{"status":"ok"}')
    return
  }
  if (url.pathname === '/messages' && request.method === 'GET') {
    const recipient = url.searchParams.get('to')?.toLowerCase()
    const result = recipient ? messages.filter((message) => message.to.toLowerCase() === recipient) : messages
    response.writeHead(200, { 'Content-Type': 'application/json' }).end(JSON.stringify(result))
    return
  }
  if (url.pathname === '/messages' && request.method === 'DELETE') {
    messages.length = 0
    response.writeHead(204).end()
    return
  }
  response.writeHead(404).end()
})

smtp.listen(12525, '127.0.0.1')
api.listen(12526, '127.0.0.1')

function shutdown() {
  smtp.close()
  api.close()
}
process.on('SIGINT', shutdown)
process.on('SIGTERM', shutdown)
