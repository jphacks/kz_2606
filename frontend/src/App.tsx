import Home from "./ui/home";
import Login from "./ui/login";

export default function App() {
  return window.location.pathname === "/login" ? <Login /> : <Home />;
}
