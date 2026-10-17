import { Navigate, Route, Routes } from 'react-router-dom'
import { AppShell } from './components/cm/AppShell'
import { Overview } from './routes/Overview'
import { Topology } from './routes/Topology'
import { Incidents } from './routes/Incidents'
import { Investigation } from './routes/Investigation'

export default function App() {
  return (
    <AppShell>
      <Routes>
        <Route path="/" element={<Overview />} />
        <Route path="/topology" element={<Topology />} />
        <Route path="/incidents" element={<Incidents />} />
        <Route path="/investigation" element={<Investigation />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </AppShell>
  )
}
