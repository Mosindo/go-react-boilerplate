import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { Text } from 'react-native';
import { ConversationsScreen } from '../features/chat/ConversationsScreen';
import { useConversations } from '../features/chat/hooks';
import { DiscoverScreen } from '../features/discover/DiscoverScreen';
import { LikesScreen } from '../features/likes/LikesScreen';
import { useLikes } from '../features/likes/hooks';
import { useNotifications } from '../features/notifications/hooks';
import { ProfileScreen } from '../features/profile/ProfileScreen';
import { fr } from '../i18n';
import { useTheme } from '../theme';
import { icons } from '../ui';
import type { MainTabParamList } from './types';

const Tab = createBottomTabNavigator<MainTabParamList>();

function glyph(char: string) {
  return function TabIcon({ color }: { color: string }) {
    return (
      <Text accessible={false} style={{ color, fontSize: 22, lineHeight: 26 }}>
        {char}
      </Text>
    );
  };
}

export function MainTabs() {
  const { colors } = useTheme();
  const { unreadTotal } = useConversations();
  const { unreadCount } = useNotifications();
  const likes = useLikes();
  const likesCount = likes.data?.length ?? 0;

  return (
    <Tab.Navigator
      screenOptions={{
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: {
          backgroundColor: colors.surface,
          borderTopColor: colors.border,
          minHeight: 56,
        },
        tabBarLabelStyle: { fontSize: 12, fontWeight: '600' },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.textOnPrimary },
      }}
      sceneContainerStyle={{ backgroundColor: colors.bg }}
    >
      <Tab.Screen
        name="Discover"
        component={DiscoverScreen}
        options={{ title: fr.tabs.discover, tabBarIcon: glyph(icons.spark) }}
      />
      <Tab.Screen
        name="Likes"
        component={LikesScreen}
        options={{
          title: fr.tabs.likes,
          tabBarIcon: glyph(icons.heart),
          tabBarBadge: likesCount > 0 ? likesCount : undefined,
          tabBarAccessibilityLabel: fr.tabs.likesA11y(likesCount),
        }}
      />
      <Tab.Screen
        name="Messages"
        component={ConversationsScreen}
        options={{
          title: fr.tabs.messages,
          tabBarIcon: glyph(icons.chat),
          tabBarBadge: unreadTotal > 0 ? unreadTotal : undefined,
          tabBarAccessibilityLabel: fr.tabs.messagesA11y(unreadTotal),
        }}
      />
      <Tab.Screen
        name="Profile"
        component={ProfileScreen}
        options={{
          title: fr.tabs.profile,
          tabBarIcon: glyph(icons.person),
          tabBarBadge: unreadCount > 0 ? unreadCount : undefined,
          tabBarAccessibilityLabel: fr.tabs.profileA11y(unreadCount),
        }}
      />
    </Tab.Navigator>
  );
}
