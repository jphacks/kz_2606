import Home from "./ui/home";
import Auth from "./ui/Auth";

export default function App() {
  const path = window.location.pathname;

  return path === "/login" || path === "/signin" ? <Auth /> : <Home />;
}
