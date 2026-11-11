import { useCallback, useEffect, useMemo, useState } from "react";
import { authApi } from "../api/services";
import { tokenStore } from "../api/client";
import { AuthContext } from "./context";

const initialState = {
  status: "loading",
  user: null,
  mustChangePassword: false,
  sessionError: null,
  expired: false,
};

function applySession(user, mustChangePassword) {
  return {
    status: "authenticated",
    user,
    mustChangePassword: Boolean(mustChangePassword),
    sessionError: null,
    expired: false,
  };
}

function stateFromSession(session) {
  if (session.status === "authenticated") {
    return applySession(session.user, session.mustChangePassword);
  }

  return {
    ...initialState,
    status: "unauthenticated",
    sessionError: session.error || null,
    expired: Boolean(session.expired),
  };
}

async function resolveSession() {
  const token = await tokenStore.load();
  if (!token) return { status: "unauthenticated" };

  try {
    const data = await authApi.me();
    const user = data?.user ?? null;
    if (!user) throw new Error("The server did not return a user.");
    return {
      status: "authenticated",
      user,
      mustChangePassword: Boolean(user.member?.must_change_password),
    };
  } catch (error) {
    if (error?.status === 401) {
      tokenStore.clear();
      return { status: "unauthenticated", expired: true };
    }
    return { status: "unauthenticated", error };
  }
}

export function AuthProvider({ children }) {
  const [state, setState] = useState(initialState);

  const loadCurrentUser = useCallback(async () => {
    const session = await resolveSession();
    setState(stateFromSession(session));
    return session.user ?? null;
  }, []);

  useEffect(() => {
    let active = true;
    resolveSession().then((session) => {
      if (active) setState(stateFromSession(session));
    });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const handleUnauthorized = () => {
      tokenStore.clear();
      setState((current) =>
        current.status === "authenticated"
          ? { ...initialState, status: "unauthenticated", expired: true }
          : current,
      );
    };

    const handlePasswordChangeRequired = () => {
      setState((current) =>
        current.status === "authenticated"
          ? { ...current, mustChangePassword: true }
          : current,
      );
    };

    window.addEventListener("lab:unauthorized", handleUnauthorized);
    window.addEventListener("lab:password-change-required", handlePasswordChangeRequired);
    return () => {
      window.removeEventListener("lab:unauthorized", handleUnauthorized);
      window.removeEventListener("lab:password-change-required", handlePasswordChangeRequired);
    };
  }, []);

  const completeLogin = useCallback(async (data, fallbackUser) => {
    if (!data?.token) {
      throw new Error("The server did not return a session token.");
    }

    await tokenStore.set(data.token);

    try {
      const me = await authApi.me();
      const user = me?.user ?? fallbackUser;
      const mustChange = Boolean(me?.user?.member?.must_change_password ?? data.must_change_password);
      setState(applySession(user, mustChange));
      return { user, mustChangePassword: mustChange };
    } catch (error) {
      if (error?.status === 401) {
        tokenStore.clear();
        setState({ ...initialState, status: "unauthenticated" });
        throw error;
      }
      setState(applySession(fallbackUser, data.must_change_password));
      return { user: fallbackUser, mustChangePassword: Boolean(data.must_change_password) };
    }
  }, []);

  const loginMember = useCallback(
    async (credentials) => {
      const data = await authApi.memberLogin(credentials);
      const fallbackUser = data?.user
        ? { ...data.user, role: data.user.role || "member" }
        : { role: "member", email: "" };
      return completeLogin(data, fallbackUser);
    },
    [completeLogin],
  );

  const loginAdmin = useCallback(
    async (credentials) => {
      const data = await authApi.adminLogin(credentials);
      const fallbackUser = data?.admin
        ? {
            id: data.admin.id,
            email: data.admin.email,
            role: "admin",
            admin: data.admin,
          }
        : { role: "admin", email: credentials.email };
      return completeLogin(data, fallbackUser);
    },
    [completeLogin],
  );

  const logout = useCallback(async () => {
    try {
      if (tokenStore.get()) await authApi.logout();
    } catch {
      // The local session is cleared even if the server is unreachable.
    } finally {
      await tokenStore.clear();
      setState({ ...initialState, status: "unauthenticated" });
    }
  }, []);

  const changePassword = useCallback(async (payload) => {
    const data = await authApi.changePassword(payload);
    setState((current) =>
      current.status === "authenticated"
        ? { ...current, mustChangePassword: false }
        : current,
    );
    return data;
  }, []);

  const refreshUser = useCallback(async () => {
    const user = await loadCurrentUser();
    return user;
  }, [loadCurrentUser]);

  const value = useMemo(() => {
    const isAuthenticated = state.status === "authenticated";
    return {
      ...state,
      isAuthenticated,
      isAdmin: Boolean(state.user?.admin || state.user?.role === "admin"),
      isMember: state.user?.role === "member",
      loginMember,
      loginAdmin,
      logout,
      changePassword,
      refreshUser,
    };
  }, [state, loginMember, loginAdmin, logout, changePassword, refreshUser]);

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}
