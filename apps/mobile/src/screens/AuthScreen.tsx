import React, { useState } from "react";
import { useAuth } from "../hooks/useAuth";
import { strings } from "../lib/strings";
import { AuthForgotScreen } from "./AuthForgotScreen";
import { AuthLoginScreen } from "./AuthLoginScreen";
import { AuthRegisterScreen } from "./AuthRegisterScreen";
import { AuthResetScreen } from "./AuthResetScreen";
import { AuthWelcomeScreen } from "./AuthWelcomeScreen";

type AuthRoute = "welcome" | "login" | "register" | "forgot" | "reset";

/** Signed-out flow: welcome, login, register, forgot password, reset password. */
export default function AuthScreen() {
  const { authNotice, clearAuthNotice } = useAuth();
  const [route, setRoute] = useState<AuthRoute>("welcome");
  const [resetDone, setResetDone] = useState(false);

  function go(next: AuthRoute) {
    clearAuthNotice();
    if (next !== "login") {
      setResetDone(false);
    }
    setRoute(next);
  }

  switch (route) {
    case "login":
      return (
        <AuthLoginScreen
          notice={resetDone ? strings.auth.resetDone : authNotice}
          onBack={() => go("welcome")}
          onForgot={() => go("forgot")}
          onRegister={() => go("register")}
        />
      );
    case "register":
      return <AuthRegisterScreen onBack={() => go("welcome")} onLogin={() => go("login")} />;
    case "forgot":
      return <AuthForgotScreen onBack={() => go("login")} onHaveToken={() => go("reset")} />;
    case "reset":
      return (
        <AuthResetScreen
          onBack={() => go("forgot")}
          onDone={() => {
            setRoute("login");
            setResetDone(true);
          }}
        />
      );
    default:
      return <AuthWelcomeScreen notice={authNotice} onLogin={() => go("login")} onRegister={() => go("register")} />;
  }
}
