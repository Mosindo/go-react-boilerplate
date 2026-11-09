import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useQuery } from "@tanstack/react-query";
import { chatApi } from "./api/endpoints";
import { useAuth } from "./auth/AuthContext";
import { keys } from "./features/keys";
import { useMyProfile } from "./features/hooks";
import { NavigationProvider, useNav, type Tab } from "./nav/Navigation";
import { useRealtime } from "./realtime/useRealtime";
import AuthScreen from "./screens/AuthScreen";
import ChatScreen from "./screens/ChatScreen";
import DiscoverScreen from "./screens/DiscoverScreen";
import MeScreen from "./screens/MeScreen";
import MessagesScreen from "./screens/MessagesScreen";
import NotificationsScreen from "./screens/NotificationsScreen";
import OnboardingScreen from "./screens/OnboardingScreen";
import ProfileDetailScreen from "./screens/ProfileDetailScreen";
import { BlockedScreen, EditPhotosScreen, EditPreferencesScreen, EditProfileScreen, PrivacyScreen } from "./screens/SettingsScreens";
import { useTheme } from "./theme/theme";
import { ErrorState, LoadingView, Text, useToast } from "./ui";
import { describeError } from "./lib/validation";

const TABS: { id: Tab; label: string; icon: string }[] = [
  { id: "discover", label: "Découvrir", icon: "♥" },
  { id: "messages", label: "Messages", icon: "✉" },
  { id: "me", label: "Profil", icon: "☺" }
];

function TabBar({ active }: { active: Tab }) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  const nav = useNav();
  const unread = useQuery({
    queryKey: keys.conversationUnread,
    queryFn: () => chatApi.conversations(),
    select: (list) => list.reduce((sum, c) => sum + c.unreadCount, 0)
  });
  return (
    <View style={[styles.tabBar, { backgroundColor: colors.surface, borderTopColor: colors.border, paddingBottom: insets.bottom + 6 }]}>
      {TABS.map((tab) => {
        const selected = tab.id === active;
        const badge = tab.id === "messages" && unread.data ? unread.data : 0;
        return (
          <Pressable
            key={tab.id}
            accessibilityRole="tab"
            accessibilityState={{ selected }}
            accessibilityLabel={`${tab.label}${badge ? `, ${badge} non lus` : ""}`}
            onPress={() => nav.goTab(tab.id)}
            style={styles.tab}
            testID={`tab-${tab.id}`}
          >
            <Text variant="heading" color={selected ? colors.primary : colors.textMuted}>
              {tab.icon}
              {badge ? ` ${badge}` : ""}
            </Text>
            <Text variant="caption" color={selected ? colors.primary : colors.textMuted}>
              {tab.label}
            </Text>
          </Pressable>
        );
      })}
    </View>
  );
}

function Routes() {
  const { route } = useNav();
  switch (route.name) {
    case "tabs":
      return (
        <View style={styles.flex}>
          <View style={styles.flex}>
            {route.tab === "discover" ? <DiscoverScreen /> : route.tab === "messages" ? <MessagesScreen /> : <MeScreen />}
          </View>
          <TabBar active={route.tab} />
        </View>
      );
    case "profile":
      return <ProfileDetailScreen userId={route.userId} />;
    case "chat":
      return <ChatScreen key={route.conversationId} route={route} />;
    case "notifications":
      return <NotificationsScreen />;
    case "editProfile":
      return <EditProfileScreen />;
    case "editPhotos":
      return <EditPhotosScreen />;
    case "editPreferences":
      return <EditPreferencesScreen />;
    case "privacy":
      return <PrivacyScreen />;
    case "blocked":
      return <BlockedScreen />;
  }
}

function SignedIn() {
  const { user } = useAuth();
  const toast = useToast();
  const me = useMyProfile();
  useRealtime(user?.id ?? null, () => toast.show("Nouveau match ! Retrouvez-le dans Messages."));

  if (me.isPending) return <LoadingView label="Chargement de votre profil…" />;
  if (me.isError) return <ErrorState message={describeError(me.error)} onRetry={() => void me.refetch()} />;
  if (!me.data || !me.data.onboardingComplete) return <OnboardingScreen profile={me.data} />;
  return (
    <NavigationProvider>
      <Routes />
    </NavigationProvider>
  );
}

export function AppShell() {
  const { status } = useAuth();
  if (status === "loading") return <LoadingView label="Chargement…" />;
  if (status === "signedOut") return <AuthScreen />;
  return <SignedIn />;
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  tabBar: { flexDirection: "row", borderTopWidth: StyleSheet.hairlineWidth, paddingTop: 8 },
  tab: { flex: 1, alignItems: "center", gap: 2, minHeight: 48 }
});
