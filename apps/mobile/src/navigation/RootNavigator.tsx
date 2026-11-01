import { createBottomTabNavigator } from "@react-navigation/bottom-tabs";
import { createNativeStackNavigator } from "@react-navigation/native-stack";
import { Ionicons } from "@expo/vector-icons";
import { Loading } from "../components/ui";
import { RealtimeProvider } from "../realtime/RealtimeProvider";
import { AuthScreen } from "../screens/AuthScreen";
import { BlockedScreen } from "../screens/BlockedScreen";
import { ChatScreen } from "../screens/ChatScreen";
import { DeleteAccountScreen } from "../screens/DeleteAccountScreen";
import { DiscoverScreen } from "../screens/DiscoverScreen";
import { EditProfileScreen } from "../screens/EditProfileScreen";
import { ForgotPasswordScreen } from "../screens/ForgotPasswordScreen";
import { MatchModalScreen } from "../screens/MatchModalScreen";
import { MessagesScreen } from "../screens/MessagesScreen";
import { NotificationsScreen } from "../screens/NotificationsScreen";
import { OnboardingScreen } from "../screens/OnboardingScreen";
import { PhotosScreen } from "../screens/PhotosScreen";
import { ProfileDetailScreen } from "../screens/ProfileDetailScreen";
import { ProfileScreen } from "../screens/ProfileScreen";
import { ReportScreen } from "../screens/ReportScreen";
import { SettingsScreen } from "../screens/SettingsScreen";
import { useAuth } from "../state/auth";
import { useConversations } from "../state/queries";
import { useColors } from "../theme";
import type { AuthStackParamList, RootStackParamList, TabParamList } from "./types";

const AuthStack = createNativeStackNavigator<AuthStackParamList>();
const RootStack = createNativeStackNavigator<RootStackParamList>();
const Tabs = createBottomTabNavigator<TabParamList>();

function MainTabs() {
  const colors = useColors();
  const conversations = useConversations();
  const unread = (conversations.data?.pages ?? [])
    .flatMap((page) => page.items)
    .reduce((sum, c) => sum + c.unreadCount, 0);

  return (
    <Tabs.Navigator
      screenOptions={({ route }) => ({
        headerShown: false,
        tabBarActiveTintColor: colors.primary,
        tabBarInactiveTintColor: colors.textMuted,
        tabBarStyle: { backgroundColor: colors.surface, borderTopColor: colors.border },
        tabBarIcon: ({ color, size, focused }) => {
          const names = {
            Discover: focused ? "flame" : "flame-outline",
            Messages: focused ? "chatbubbles" : "chatbubbles-outline",
            Me: focused ? "person" : "person-outline",
          } as const;
          return <Ionicons color={color} name={names[route.name]} size={size} />;
        },
      })}
    >
      <Tabs.Screen component={DiscoverScreen} name="Discover" options={{ title: "Découvrir" }} />
      <Tabs.Screen
        component={MessagesScreen}
        name="Messages"
        options={{ title: "Messages", tabBarBadge: unread > 0 ? unread : undefined }}
      />
      <Tabs.Screen component={ProfileScreen} name="Me" options={{ title: "Profil" }} />
    </Tabs.Navigator>
  );
}

function MainStack() {
  return (
    <RealtimeProvider>
      <RootStack.Navigator>
        <RootStack.Screen component={MainTabs} name="Tabs" options={{ headerShown: false }} />
        <RootStack.Screen component={ChatScreen} name="Chat" options={{ title: "" }} />
        <RootStack.Screen component={ProfileDetailScreen} name="ProfileDetail" options={{ title: "Profil" }} />
        <RootStack.Screen component={EditProfileScreen} name="EditProfile" options={{ title: "Modifier le profil" }} />
        <RootStack.Screen component={PhotosScreen} name="Photos" options={{ title: "Mes photos" }} />
        <RootStack.Screen component={SettingsScreen} name="Settings" options={{ title: "Préférences" }} />
        <RootStack.Screen component={BlockedScreen} name="Blocked" options={{ title: "Profils bloqués" }} />
        <RootStack.Screen component={NotificationsScreen} name="Notifications" options={{ title: "Notifications" }} />
        <RootStack.Screen
          component={ReportScreen}
          name="Report"
          options={{ title: "Signaler", presentation: "modal" }}
        />
        <RootStack.Screen
          component={DeleteAccountScreen}
          name="DeleteAccount"
          options={{ title: "Supprimer mon compte" }}
        />
        <RootStack.Screen
          component={MatchModalScreen}
          name="MatchModal"
          options={{ headerShown: false, presentation: "transparentModal", animation: "fade" }}
        />
      </RootStack.Navigator>
    </RealtimeProvider>
  );
}

export function RootNavigator() {
  const { user, isBooting } = useAuth();

  if (isBooting) {
    return <Loading label="Chargement…" />;
  }
  if (!user) {
    return (
      <AuthStack.Navigator screenOptions={{ headerShown: false }}>
        <AuthStack.Screen component={AuthScreen} name="Auth" />
        <AuthStack.Screen
          component={ForgotPasswordScreen}
          name="ForgotPassword"
          options={{ headerShown: true, title: "Mot de passe oublié" }}
        />
      </AuthStack.Navigator>
    );
  }
  if (!user.profileComplete) {
    return <OnboardingScreen />;
  }
  return <MainStack />;
}
