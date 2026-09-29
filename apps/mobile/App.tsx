import React, { useCallback, useEffect, useState } from "react";
import { QueryClientProvider } from "@tanstack/react-query";
import { StatusBar } from "expo-status-bar";
import { View } from "react-native";
import { SafeAreaProvider } from "react-native-safe-area-context";
import { AuthProvider, useAuth } from "./src/hooks/useAuth";
import { queryClient } from "./src/hooks/queryClient";
import { RealtimeProvider, useBadges } from "./src/realtime/RealtimeProvider";
import AuthScreen from "./src/screens/AuthScreen";
import ConversationsScreen from "./src/screens/ConversationsScreen";
import DiscoverScreen from "./src/screens/DiscoverScreen";
import NotificationsScreen from "./src/screens/NotificationsScreen";
import OnboardingScreen from "./src/screens/OnboardingScreen";
import ProfileScreen from "./src/screens/ProfileScreen";
import { clearGlobalError, ErrorView, LoadingView, ToastHost, useGlobalFeedback } from "./src/shared/feedback";
import { SafeAreaLayout, TabBar, type TabItem } from "./src/shared/layout";
import { ThemeProvider, useTheme, useThemedStyles, type Theme } from "./src/shared/ui";

type TabKey = "discover" | "chats" | "notifications" | "profile";

const makeStyles = (t: Theme) => ({
  root: { flex: 1, backgroundColor: t.colors.background },
  content: { flex: 1 },
  bannerWrap: {
    position: "absolute" as const,
    left: t.spacing.lg,
    right: t.spacing.lg,
    bottom: t.spacing.xxxl + t.spacing.xxl
  },
  bootError: { padding: t.spacing.lg, gap: t.spacing.md, flex: 1, justifyContent: "center" as const }
});

function MainTabs() {
  const styles = useThemedStyles(makeStyles);
  const badges = useBadges();
  const [tab, setTab] = useState<TabKey>("discover");
  const [openConversationId, setOpenConversationId] = useState<string | null>(null);

  const openConversation = useCallback((conversationId: string) => {
    setOpenConversationId(conversationId);
    setTab("chats");
  }, []);
  const clearOpenConversation = useCallback(() => setOpenConversationId(null), []);

  const items: readonly TabItem<TabKey>[] = [
    { key: "discover", label: "Discover", glyph: "✦" },
    { key: "chats", label: "Chats", glyph: "✉︎", badge: badges.unreadMessages },
    { key: "notifications", label: "Notifications", glyph: "◉", badge: badges.unreadNotifications },
    { key: "profile", label: "Profile", glyph: "☺︎" }
  ];

  return (
    <View style={styles.root}>
      <View style={styles.content}>
        {tab === "discover" ? <DiscoverScreen onOpenConversation={openConversation} /> : null}
        {tab === "chats" ? (
          <ConversationsScreen
            onOpenedConversation={clearOpenConversation}
            openConversationId={openConversationId}
          />
        ) : null}
        {tab === "notifications" ? <NotificationsScreen onOpenConversation={openConversation} /> : null}
        {tab === "profile" ? <ProfileScreen /> : null}
      </View>
      <TabBar active={tab} items={items} onChange={setTab} />
    </View>
  );
}

function AppShell() {
  const styles = useThemedStyles(makeStyles);
  const theme = useTheme();
  const { bootError, isAuthenticated, isBooting, retryBoot, user } = useAuth();
  const globalFeedback = useGlobalFeedback();

  useEffect(() => {
    if (!globalFeedback.error) {
      return;
    }
    const timeoutId = setTimeout(clearGlobalError, 5000);
    return () => clearTimeout(timeoutId);
  }, [globalFeedback.error]);

  let body: React.ReactNode;
  if (isBooting) {
    body = (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Restoring your session..." />
      </SafeAreaLayout>
    );
  } else if (bootError && !isAuthenticated) {
    body = (
      <SafeAreaLayout>
        <View style={styles.bootError}>
          <ErrorView
            actionLabel="Try again"
            message={bootError}
            onAction={retryBoot}
            testID="boot-error"
            title="Cannot reach Alba"
          />
        </View>
      </SafeAreaLayout>
    );
  } else if (!isAuthenticated || !user) {
    body = <AuthScreen />;
  } else if (!user.profileComplete) {
    body = <OnboardingScreen key={user.id} />;
  } else {
    body = (
      <RealtimeProvider key={user.id}>
        <SafeAreaLayout edges={["top", "left", "right"]}>
          <MainTabs />
        </SafeAreaLayout>
      </RealtimeProvider>
    );
  }

  return (
    <View style={styles.root}>
      <StatusBar style={theme.isDark ? "light" : "dark"} />
      {body}
      {globalFeedback.error ? (
        <View pointerEvents="box-none" style={styles.bannerWrap}>
          <ErrorView
            actionLabel="Dismiss"
            compact
            message={globalFeedback.error}
            onAction={clearGlobalError}
            title="Request issue"
          />
        </View>
      ) : null}
      <ToastHost />
    </View>
  );
}

export default function App() {
  return (
    <SafeAreaProvider>
      <ThemeProvider>
        <QueryClientProvider client={queryClient}>
          <AuthProvider>
            <AppShell />
          </AuthProvider>
        </QueryClientProvider>
      </ThemeProvider>
    </SafeAreaProvider>
  );
}
