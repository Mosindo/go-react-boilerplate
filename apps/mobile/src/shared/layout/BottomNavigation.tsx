import React from "react";
import { Platform, StyleSheet } from "react-native";
import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import ConversationsScreen from "../../screens/ConversationsScreen";
import HomeScreen from "../../screens/HomeScreen";
import NotificationsScreen from "../../screens/NotificationsScreen";
import ProfileScreen from "../../screens/ProfileScreen";
import type { TabParamList } from "../../navigation/types";
import { useConversations, useNotifications } from "../../hooks/queries";
import { Text, colors, shadows, spacing } from "../ui";

const Tab = createBottomTabNavigator<TabParamList>();

function TabGlyph({ glyph, focused }: { glyph: string; focused: boolean }) {
  return (
    <Text style={[styles.glyph, { color: focused ? colors.primary : colors.textMuted }]} weight="bold">
      {glyph}
    </Text>
  );
}

/** Main tab bar: discovery, conversations, notifications and the user's own profile. */
export function BottomNavigation() {
  const conversations = useConversations();
  const notifications = useNotifications();
  const unreadMessages = conversations.data?.pages[0]?.unreadTotal ?? 0;
  const unreadNotifications = notifications.data?.pages[0]?.unreadCount ?? 0;

  return (
    <Tab.Navigator
      initialRouteName="Home"
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarHideOnKeyboard: true,
        tabBarStyle: styles.bar,
        tabBarLabelStyle: styles.label,
        tabBarItemStyle: styles.item
      }}
      sceneContainerStyle={{ backgroundColor: colors.background }}
    >
      <Tab.Screen
        component={HomeScreen}
        name="Home"
        options={{
          title: "Découvrir",
          tabBarTestID: "tab-home",
          tabBarAccessibilityLabel: "Découvrir",
          tabBarIcon: ({ focused }) => <TabGlyph focused={focused} glyph="♥" />
        }}
      />
      <Tab.Screen
        component={ConversationsScreen}
        name="Messages"
        options={{
          title: "Messages",
          tabBarTestID: "tab-messages",
          tabBarAccessibilityLabel: unreadMessages > 0 ? `Messages, ${unreadMessages} non lus` : "Messages",
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarBadgeStyle: styles.badge,
          tabBarIcon: ({ focused }) => <TabGlyph focused={focused} glyph="✉" />
        }}
      />
      <Tab.Screen
        component={NotificationsScreen}
        name="Notifications"
        options={{
          title: "Activité",
          tabBarTestID: "tab-notifications",
          tabBarAccessibilityLabel: unreadNotifications > 0 ? `Activité, ${unreadNotifications} non lues` : "Activité",
          tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined,
          tabBarBadgeStyle: styles.badge,
          tabBarIcon: ({ focused }) => <TabGlyph focused={focused} glyph="●" />
        }}
      />
      <Tab.Screen
        component={ProfileScreen}
        name="Profile"
        options={{
          title: "Profil",
          tabBarTestID: "tab-profile",
          tabBarAccessibilityLabel: "Profil",
          tabBarIcon: ({ focused }) => <TabGlyph focused={focused} glyph="☺" />
        }}
      />
    </Tab.Navigator>
  );
}

// Only the web build needs the explicit height: native tab bars size themselves.
const styles = StyleSheet.create({
  bar: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    bottom: spacing.md,
    height: Platform.OS === "web" ? 64 : 68,
    paddingTop: spacing.xs,
    backgroundColor: colors.backgroundElevated,
    borderTopColor: colors.border,
    borderTopWidth: 1,
    borderRadius: 22,
    ...shadows.floating
  },
  item: { paddingVertical: spacing.xs },
  label: { paddingBottom: spacing.xs, fontSize: 12, fontWeight: "600" },
  glyph: { fontSize: 22, lineHeight: 26 },
  badge: { backgroundColor: colors.primary, color: colors.inverse, fontSize: 11, fontWeight: "700" }
});
