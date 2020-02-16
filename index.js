import { observable } from 'mobx'
import { observer } from 'mobx-react-lite'
import React, { useRef, useEffect } from 'react'
import ReactDOM from 'react-dom'

let state = observable({
  messages: [],
  text: '',
})

let wsOrigin = window.location.origin.split(':').slice(1).join(':')
let ws = new WebSocket(`ws://${wsOrigin}/sock`)
ws.onmessage = function ({ data }) {
  state.messages.push(data)
}

function sendMessage (e) {
  e.preventDefault()
  ws.send(state.text)
  state.text = ''
}

function updateText ({ target }) {
  state.text = target.value
}

let App = observer(function App () {
  let output = useRef(null)

  useEffect(function () {
    let el = output.current
    el.scrollTop = el.scrollHeight
  })

  return (
    <>
      <div style={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: 'space-between',
      }}>
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
          }}>
          {state.messages.map(function (message) {
            return (
              <li>
                <pre>
                  {message}
                </pre>
              </li>
            )
          })}
          <li>
            <form
              style={{
                background: 'rgba(0, 0, 0, 0.1)',
                padding: '0.2rem 0.5rem',
                fontFamily: 'monospace',
                display: 'flex',
                justifyContent: 'space-between',
                alignItems: 'center',
              }}
              onSubmit={sendMessage}>
              >
              <input
                autoFocus
                value={state.text}
                style={{
                  width: '100%',
                  border: 0,
                  outline: 0,
                  fontFamily: 'monospace',
                  background: 'transparent',
                }}
                onChange={updateText}/>
            </form>
          </li>
        </ul>
      </div>
    </>
  )
})

ReactDOM.render(<App/>, document.getElementById('app'))
