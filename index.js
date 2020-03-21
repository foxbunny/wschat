import { observable, runInAction } from 'mobx'
import { observer } from 'mobx-react-lite'
import React, { useEffect, useRef } from 'react'
import ReactDOM from 'react-dom'

// -----------------------------------------------------------------------------
// CONSTANTS
// -----------------------------------------------------------------------------

const MAX_MESSAGE_LENGTH = 47
const DEFAULT_FREQUENCY = 1000
const DEFAULT_BANDWIDTH = 400
const DEFAULT_CODING_RATE = 5
const DEFAULT_SPREADING_FACTOR = 12

const BANDWIDTH_OPTIONS = [
  [200, 200],
  [400, 400],
  [800, 800],
  [1600, 1600],
]
const SPREADING_FACTOR_OPTIONS = [
  [5, 5],
  [6, 6],
  [7, 7],
  [8, 8],
  [9, 9],
  [10, 10],
  [11, 11],
  [12, 12],
]
const CODING_RATE_OPTIONS = [
  ['4/5', 5],
  ['4/6', 6],
  ['4/7', 7],
  ['4/8', 8],
]
const ORIGIN = window.location.origin.split(':').slice(1).join(':')

const ME = Symbol('me')
const SYSTEM = Symbol('system')
const ALIASES = new Map([
  [ME, 'Me'],
  [SYSTEM, 'System'],
])

// -----------------------------------------------------------------------------
// THEME
// -----------------------------------------------------------------------------

const THEME = {
  fontFamily: 'Helvetica, Arial, sans-serif',
  fontSize: '16px',
  clickableElementColor: '#00b2df',
  chatAreaBackground: '#efefef',
  chatFontSize: '18px',
}

const BUTTON_STYLE = {
  fontFamily: THEME.fontFamily,
  fontSize: THEME.fontSize,
  padding: '0.2rem 1rem',
  background: THEME.clickableElementColor,
  border: 0,
  cursor: 'pointer',
  borderRadius: '3px',
  color: 'white',
}

const INPUT_STYLE = {
  fontFamily: THEME.fontFamily,
  fontSize: THEME.fontSize,
  padding: '0.2rem',
  border: '1px solid #555',
  width: '100%',
  borderRadius: '3px',
}

const MESSAGE_BACKGROUND = {
  background: '#d7e9a7',
  border: '1px solid #bbcb91',
  borderRadius: '3px',
  marginBottom: '1rem',
  padding: '1rem',
  width: '80vw',
}

const SPECIAL_MESSAGES = new Map([
  [
    SYSTEM,
    {
      background: '#000',
      border: '1px solid #000',
      color: 'white',
      marginLeft: 'auto',
    },
  ],
  [
    ME,
    {
      marginLeft: 'auto',
      background: '#a9daeb',
      border: '1px solid #96c1d0',
    },
  ],
])

const CALLSIGN_STYLE = {
  fontWeight: 'bold',
}

// -----------------------------------------------------------------------------
// APPLICATION STATE
// -----------------------------------------------------------------------------

function parseMessage (text) {
  let m = /^\[([^\]]+)]:(.*)$/.exec(text)
  if (!m) return [SYSTEM, text]
  return [m[1], m[2].trim()]
}

let state = observable({
  callsign: localStorage.callsign || 'Anonymous',
  params: {
    frequency: '' + DEFAULT_FREQUENCY,
    bandwidth: '' + DEFAULT_BANDWIDTH,
    spreadingFactor: '' + DEFAULT_SPREADING_FACTOR,
    codingRate: '' + DEFAULT_CODING_RATE,
  },
  messages: [],
  text: '',
  charCount: 0,
  socket: null,

  get charsRemaining () {
    return MAX_MESSAGE_LENGTH - this.charCount
  },

  get isConnected () {
    return this.socket != null
  },

  get freqError () {
    let freq = parseFloat(this.params.frequency)
    return !isNaN(freq) && freq >= 40 && freq <= 6000
      ? null
      : 'Valid frequency range is from 40MHz through 6000MHz'
  },

  updateText (text) {
    let nChars = text.length
    if (text.length > MAX_MESSAGE_LENGTH) return
    this.charCount = nChars
    this.text = text
  },

  updateParam (paramName, value) {
    this.params[paramName] = value
  },

  connect () {
    let model = this
    localStorage.callsign = model.callsign
    let q = []
    for (let [param, value] of Object.entries(model.params)) {
      q.push(`${param}=${encodeURIComponent(value)}`)
    }
    let ws = new WebSocket(`ws://${ORIGIN}/sock?${q.join('&')}`)
    ws.onmessage = function ({ data }) {
      if (data.startsWith('>')) data = data.slice(1)
      let [callsign, text] = parseMessage(data)
      if (text) model.addMessage(callsign, text)
    }
    ws.onclose = function () {
      model.socket = null
    }
    ws.onerror = function () {
      alert('Connection error. Server may be offline.')
    }
    ws.onopen = function () {
      model.socket = ws
    }
  },

  disconnect () {
    this.socket.close()
  },

  send () {
    let model = this
    if (this.socket) {
      this.socket.send(`[${this.callsign}]: ${this.text}`)
      runInAction(function () {
        model.addMessage(ME, model.text)
        model.text = ''
      })
    }
  },

  addMessage (callsign, text) {
    let lastMessage = this.messages[this.messages.length - 1]
    if (lastMessage?.callsign === callsign) lastMessage.text += '\n' + text
    else this.messages.push({ callsign, text })
  },
})

// -----------------------------------------------------------------------------
// COMPONENTS
// -----------------------------------------------------------------------------

let Select = observer(function ({ label, options, param }) {
  function onChange (e) {
    state.updateParam(param, e.target.value)
  }

  return (
    <div style={{ marginBottom: '1rem' }}>
      <label style={{ display: 'block', marginBottom: '0.5rem' }}>
        <p>{label}:</p>
        <select style={INPUT_STYLE} onChange={onChange}
                value={state.params[param]}>
          {options.map(function ([label, value]) {
            return <option value={value}>{label}</option>
          })}
        </select>
      </label>
    </div>
  )
})

let Input = observer(function ({ label, error, param, property }) {
  function onChange (e) {
    if (param) state.updateParam(param, e.target.value)
    else if (property) state[property] = e.target.value
  }

  let value = param ? state.params[param] : state[property]

  return (
    <div style={{ marginBottom: '1pt' }}>
      <label style={{ display: 'block', marginBottom: '0.5rem' }}>
        <p>{label}:</p>
        <input style={INPUT_STYLE} type="numeric" value={value}
               onChange={onChange}/>

      </label>
      {error && <p style={{ color: 'red' }}>{error}</p>}
    </div>
  )
})

let LinkButton = observer(function ({ children, onClick }) {
  return (
    <button
      onClick={onClick}
      style={{
        textDecoration: 'underline',
        border: 0,
        background: 'transparent',
        cursor: 'pointer',
        color: THEME.clickableElementColor,
        fontFamily: THEME.fontFamily,
        fontSize: THEME.fontSize,
      }}>
      {children}
    </button>
  )
})

let Message = observer(function ({ message }) {
  let messageStyle = {
    ...MESSAGE_BACKGROUND,
    ...SPECIAL_MESSAGES.get(message.callsign),
  }

  let callsign = ALIASES.get(message.callsign) || message.callsign

  return (
    <li style={messageStyle}>
      <p style={CALLSIGN_STYLE}>{callsign}</p>
      <pre style={{ fontFamily: THEME.fontFamily }}>
        {message.text}
      </pre>
    </li>
  )
})

// -----------------------------------------------------------------------------
// PAGES
// -----------------------------------------------------------------------------

let Setup = observer(function App () {
  function onSubmit (e) {
    e.preventDefault()
    state.connect()
  }

  return (
    <div style={{
      display: 'flex',
      justifyContent: 'center',
      alignItems: 'center',
      fontFamily: THEME.fontFamily,
      fontSize: THEME.fontSize,
    }}>
      <div>
        <h1 style={{
          marginBottom: '2rem',
          textAlign: 'center',
          fontWeight: 'normal',
        }}>
          Othernet radio chat
        </h1>
        <form onSubmit={onSubmit}>
          <Input label="Callsign/name" property="callsign"/>
          <Input label="Frequency (MHz)" param="frequency"
                 error={state.freqError}/>
          <Select label="Bandwdith (kHz)" options={BANDWIDTH_OPTIONS}
                  param="bandwidth"/>
          <Select label="Spreading factor" options={SPREADING_FACTOR_OPTIONS}
                  param="spreadingFactor"/>
          <Select label="Coding rate" options={CODING_RATE_OPTIONS}
                  param="codingRate"/>
          <button style={BUTTON_STYLE}>Connect</button>
        </form>
      </div>
    </div>
  )
})

let ChatWindow = observer(function () {
  let output = useRef(null)

  useEffect(function () {
    let el = output.current
    el.scrollTop = el.scrollHeight
  })

  function sendMessage (e) {
    e.preventDefault()
    state.send()
  }

  function updateText (e) {
    state.updateText(e.target.value)
  }

  function disconnect () {
    let confirmed = confirm('Do you to disconnect from the chat session?')
    if (confirmed) state.disconnect()
  }

  return (
    <>
      <div style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
        fontFamily: THEME.fontFamily,
        fontSize: THEME.fontSize,
      }}>
        <div
          style={{
            padding: '0.5rem 0',
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}>
          <button style={BUTTON_STYLE} onClick={disconnect}>
            Disconnect
          </button>
          <LinkButton onClick={disconnect}>
            Freq: {state.params.frequency} MHz
          </LinkButton>
          <LinkButton onClick={disconnect}>
            BW: {state.params.bandwidth} kHz
          </LinkButton>
          <LinkButton onClick={disconnect}>
            SF: {state.params.spreadingFactor}
          </LinkButton>
          <LinkButton onClick={disconnect}>
            CR: 4 / {state.params.codingRate}
          </LinkButton>
        </div>
        <ul
          ref={output}
          style={{
            boxShadow: 'inset 5px 5px 10px rgba(0, 0, 0, 0.2)',
            height: '100%',
            overflowY: 'auto',
            borderTop: '1px solid #ddd',
            borderLeft: '1px solid #ddd',
            borderRight: '1px solid white',
            borderBottom: '1px solid white',
            padding: '1rem',
            marginBottom: '1rem',
          }}>
          {state.messages.map(function (message) {
            return (
              <Message message={message}/>
            )
          })}
        </ul>
        <form
          style={{
            background: 'white',
            fontFamily: THEME.chatFontFamily,
            fontSize: THEME.chatFontSize,
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            height: '3rem',
            boxShadow: 'inset 3px 3px 5px rgba(0, 0, 0, 0.2)',
            border: '1px solid #ddd',
            borderRadius: '5px',
            overflow: 'hidden',
          }}
          onSubmit={sendMessage}>
          <input
            autoFocus
            value={state.text}
            style={{
              padding: '0.2rem 0.5rem',
              width: '100%',
              border: 0,
              outline: 0,
              fontFamily: THEME.fontFamily,
              fontSize: THEME.fontSize,
              background: 'transparent',
            }}
            onChange={updateText}/>
          <span style={{
            color: state.charsRemaining ? '#999' : 'red',
            fontSize: '0.8rem',
            padding: '0 1rem',
            textAlign: 'right',
            width: '4rem',
          }}>
            {state.charsRemaining}
          </span>
          <button style={{
            padding: '0.2rem 0.5rem',
            background: THEME.clickableElementColor,
            color: 'white',
            height: '100%',
            width: '8rem',
            cursor: 'pointer',
            border: '0',
            fontFamily: THEME.fontFamily,
            fontSize: THEME.fontSize,
            borderRadius: '0',
          }}>
            Send ↵
          </button>
        </form>
      </div>
    </>
  )
})

// -----------------------------------------------------------------------------
// MAIN APP
// -----------------------------------------------------------------------------

let App = observer(function App () {
  if (state.socket == null) return <Setup/>
  return <ChatWindow/>
})

ReactDOM.render(<App/>, document.getElementById('app'))
