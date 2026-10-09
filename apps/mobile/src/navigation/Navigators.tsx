import React from 'react';
import { Platform } from 'react-native';
import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { Ionicons } from '@expo/vector-icons';
import { fr } from '../i18n/fr';
import { useTheme } from '../theme';
import { useUnreadMessages, useUnreadNotifications } from '../query/hooks';
import { IconButton } from '../shared/ui';
import { WelcomeScreen } from '../screens/auth/WelcomeScreen';
import { LoginScreen } from '../screens/auth/LoginScreen';
import { RegisterScreen } from '../screens/auth/RegisterScreen';
import { ForgotPasswordScreen } from '../screens/auth/ForgotPasswordScreen';
import { ResetPasswordScreen } from '../screens/auth/ResetPasswordScreen';
import { DiscoverScreen } from '../screens/discover/DiscoverScreen';
import { LikesScreen } from '../screens/likes/LikesScreen';
import { MessagesScreen } from '../screens/messages/MessagesScreen';
import { ChatScreen } from '../screens/messages/ChatScreen';
import { ProfileScreen } from '../screens/profile/ProfileScreen';
import { MyPreviewScreen } from '../screens/profile/MyPreviewScreen';
import { EditProfileScreen } from '../screens/profile/EditProfileScreen';
import { EditPhotosScreen } from '../screens/profile/EditPhotosScreen';
import { EditPreferencesScreen } from '../screens/profile/EditPreferencesScreen';
import { PrivacyScreen } from '../screens/profile/PrivacyScreen';
import { NotificationsScreen } from '../screens/profile/NotificationsScreen';
import { BlockedScreen } from '../screens/profile/BlockedScreen';
import { ChangePasswordScreen } from '../screens/profile/ChangePasswordScreen';
import { DeleteAccountScreen } from '../screens/profile/DeleteAccountScreen';
import { ProfileDetailScreen } from '../screens/ProfileDetailScreen';
import type { AuthStackParamList, MainStackParamList, TabParamList } from './types';

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const MainStack = createNativeStackNavigator<MainStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

export function AuthNavigator() {
  const { colors } = useTheme();
  return (
    <AuthStack.Navigator
      screenOptions={{
        headerShadowVisible: false,
        headerTitle: '',
        headerBackTitle: fr.common.back,
        headerTintColor: colors.text,
        headerStyle: { backgroundColor: colors.background },
        contentStyle: { backgroundColor: colors.background },
      }}
    >
      <AuthStack.Screen name="Welcome" component={WelcomeScreen} options={{ headerShown: false }} />
      <AuthStack.Screen name="Login" component={LoginScreen} />
      <AuthStack.Screen name="Register" component={RegisterScreen} />
      <AuthStack.Screen name="ForgotPassword" component={ForgotPasswordScreen} />
      <AuthStack.Screen name="ResetPassword" component={ResetPasswordScreen} />
    </AuthStack.Navigator>
  );
}

function TabsNavigator() {
  const { colors } = useTheme();
  const unreadMessages = useUnreadMessages();
  const unreadNotifications = useUnreadNotifications();
  const icon = (
    name: keyof typeof Ionicons.glyphMap,
    focusedName: keyof typeof Ionicons.glyphMap,
  ) =>
    function TabIcon({ color, size, focused }: { color: string; size: number; focused: boolean }) {
      return <Ionicons name={focused ? focusedName : name} size={size} color={color} />;
    };
  return (
    <Tabs.Navigator
      screenOptions={{
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarBadgeStyle: { backgroundColor: colors.primary, color: colors.onPrimary },
        headerStyle: { backgroundColor: colors.background },
        headerShadowVisible: false,
        headerTitleStyle: { color: colors.text, fontWeight: '800' },
        sceneStyle: { backgroundColor: colors.background },
      }}
    >
      <Tabs.Screen
        name="Discover"
        component={DiscoverScreen}
        options={({ navigation }) => ({
          title: fr.tabs.discover,
          tabBarButtonTestID: 'tab-discover',
          tabBarAccessibilityLabel: fr.tabs.discover,
          tabBarIcon: icon('compass-outline', 'compass'),
          headerRight: () => (
            <IconButton
              icon="notifications-outline"
              label={`${fr.profile.notifications}${unreadNotifications ? ` (${fr.tabsA11y.unread(unreadNotifications)})` : ''}`}
              badge={unreadNotifications}
              onPress={() => navigation.getParent()?.navigate('Notifications' as never)}
              testID="discover-bell"
            />
          ),
        })}
      />
      <Tabs.Screen
        name="Likes"
        component={LikesScreen}
        options={{
          title: fr.tabs.likes,
          tabBarButtonTestID: 'tab-likes',
          tabBarAccessibilityLabel: fr.tabs.likes,
          tabBarIcon: icon('heart-outline', 'heart'),
        }}
      />
      <Tabs.Screen
        name="Messages"
        component={MessagesScreen}
        options={{
          title: fr.tabs.messages,
          tabBarButtonTestID: 'tab-messages',
          tabBarAccessibilityLabel: unreadMessages
            ? `${fr.tabs.messages}, ${fr.tabsA11y.unread(unreadMessages)}`
            : fr.tabs.messages,
          tabBarBadge: unreadMessages > 0 ? unreadMessages : undefined,
          tabBarIcon: icon('chatbubble-outline', 'chatbubble'),
        }}
      />
      <Tabs.Screen
        name="Profile"
        component={ProfileScreen}
        options={{
          title: fr.tabs.profile,
          tabBarButtonTestID: 'tab-profile',
          tabBarAccessibilityLabel: fr.tabs.profile,
          tabBarBadge: unreadNotifications > 0 ? unreadNotifications : undefined,
          tabBarIcon: icon('person-outline', 'person'),
        }}
      />
    </Tabs.Navigator>
  );
}

export function MainNavigator() {
  const { colors } = useTheme();
  return (
    <MainStack.Navigator
      screenOptions={{
        headerShadowVisible: false,
        headerTintColor: colors.text,
        headerBackTitle: fr.common.back,
        headerStyle: { backgroundColor: colors.background },
        contentStyle: { backgroundColor: colors.background },
        animation: Platform.OS === 'web' ? 'none' : 'default',
      }}
    >
      <MainStack.Screen name="Tabs" component={TabsNavigator} options={{ headerShown: false }} />
      <MainStack.Screen
        name="ProfileDetail"
        component={ProfileDetailScreen}
        options={{ title: '' }}
      />
      <MainStack.Screen name="Chat" component={ChatScreen} options={{ title: '' }} />
      <MainStack.Screen
        name="MyPreview"
        component={MyPreviewScreen}
        options={{ title: fr.profile.preview }}
      />
      <MainStack.Screen
        name="EditProfile"
        component={EditProfileScreen}
        options={{ title: fr.profile.editTitle }}
      />
      <MainStack.Screen
        name="EditPhotos"
        component={EditPhotosScreen}
        options={{ title: fr.photos.edit }}
      />
      <MainStack.Screen
        name="EditPreferences"
        component={EditPreferencesScreen}
        options={{ title: fr.preferences.title }}
      />
      <MainStack.Screen
        name="Privacy"
        component={PrivacyScreen}
        options={{ title: fr.profile.privacyTitle }}
      />
      <MainStack.Screen
        name="Notifications"
        component={NotificationsScreen}
        options={{ title: fr.notifications.title }}
      />
      <MainStack.Screen
        name="Blocked"
        component={BlockedScreen}
        options={{ title: fr.safety.blockedListTitle }}
      />
      <MainStack.Screen
        name="ChangePassword"
        component={ChangePasswordScreen}
        options={{ title: fr.account.changePasswordTitle }}
      />
      <MainStack.Screen
        name="DeleteAccount"
        component={DeleteAccountScreen}
        options={{ title: fr.account.deleteTitle }}
      />
    </MainStack.Navigator>
  );
}
