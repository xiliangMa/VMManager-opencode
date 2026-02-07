import React from 'react'
import { useParams } from 'react-router-dom'

const VMConsole: React.FC = () => {
  const { id } = useParams()
  return <div>VM Console: {id}</div>
}

export default VMConsole
