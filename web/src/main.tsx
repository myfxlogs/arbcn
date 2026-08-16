import { StrictMode } from "react";
import { createRoot } from "react-dom/client";

import App from "./App";
import "./styles/00-base.css";
import "./styles/01-header-card.css";
import "./styles/02-collapse.css";
import "./styles/03-table-chip.css";
import "./styles/04-freshness-opp.css";
import "./styles/05-insights-review.css";
import "./styles/06-buttons.css";
import "./styles/07-bell-tabs.css";
import "./styles/08-facts-ledger.css";
import "./styles/09-sim.css";
import "./styles/10-testnet-quote.css";
import "./styles/11-account-responsive.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
