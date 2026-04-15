import { Navigate, Route, Routes } from "react-router-dom";
import { ConsoleShell } from "./components/skm/ConsoleShell";
import { NotFoundPage } from "./pages/NotFoundPage";
import { AppLayout } from "./components/AppLayout";
import { Toaster } from "./components/ui/sonner";
import { DashboardPage } from "./pages/DashboardPage";
import { IssuesPage } from "./pages/IssuesPage";
import { ProvidersPage } from "./pages/ProvidersPage";
import { SkillsPage } from "./pages/SkillsPage";

function App() {
  return (
    <AppLayout>
      <Routes>
        <Route element={<ConsoleShell />}>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<DashboardPage />} />
          <Route path="/skills" element={<SkillsPage />} />
          <Route path="/skills/:zid" element={<SkillsPage />} />
          <Route path="/providers" element={<ProvidersPage />} />
          <Route path="/issues" element={<IssuesPage />} />
        </Route>
        <Route path="*" element={<NotFoundPage />} />
      </Routes>

      <Toaster richColors />
    </AppLayout>
  );
}

export default App;
