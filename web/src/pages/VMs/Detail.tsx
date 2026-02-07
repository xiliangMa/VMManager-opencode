import React from 'react'
import { useParams } from 'react-router-dom'

const VMDetail: React.FC = () => {
  const { id } = useParams()
  return <div>VM Detail: {id}</div>
}

export default VMDetail
