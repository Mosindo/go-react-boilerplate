import { createNativeStackNavigator } from '@react-navigation/native-stack';
import { ChatScreen } from '../features/chat/ChatScreen';
import { DeckProvider } from '../features/discover/DeckContext';
import { MatchProvider } from '../features/discover/MatchContext';
import { NotificationsScreen } from '../features/notifications/NotificationsScreen';
import { ChangePasswordScreen } from '../features/profile/ChangePasswordScreen';
import { DeleteAccountScreen } from '../features/profile/DeleteAccountScreen';
import { EditProfileScreen } from '../features/profile/EditProfileScreen';
import { PhotosScreen } from '../features/profile/PhotosScreen';
import { PreferencesScreen } from '../features/profile/PreferencesScreen';
import { PrivacyScreen } from '../features/profile/PrivacyScreen';
import { ProfileDetailScreen } from '../features/profile/ProfileDetailScreen';
import { BlockedUsersScreen } from '../features/safety/BlockedUsersScreen';
import { ReportScreen } from '../features/safety/ReportScreen';
import { fr } from '../i18n';
import { RealtimeProvider } from '../realtime/RealtimeProvider';
import { MainTabs } from './MainTabs';
import { navigationRef } from './ref';
import type { RootStackParamList } from './types';
import { useStackOptions } from './useStackOptions';

const Stack = createNativeStackNavigator<RootStackParamList>();

export function MainNavigator() {
  const options = useStackOptions();
  return (
    <DeckProvider>
      <MatchProvider navigationRef={navigationRef}>
        <RealtimeProvider navigationRef={navigationRef}>
          <Stack.Navigator screenOptions={options}>
            <Stack.Screen name="Tabs" component={MainTabs} options={{ headerShown: false }} />
            <Stack.Screen name="Chat" component={ChatScreen} />
            <Stack.Screen
              name="ProfileDetail"
              component={ProfileDetailScreen}
              options={{ title: '' }}
            />
            <Stack.Screen
              name="Notifications"
              component={NotificationsScreen}
              options={{ title: fr.nav.notifications }}
            />
            <Stack.Screen
              name="EditProfile"
              component={EditProfileScreen}
              options={{ title: fr.nav.editProfile }}
            />
            <Stack.Screen
              name="Photos"
              component={PhotosScreen}
              options={{ title: fr.nav.photos }}
            />
            <Stack.Screen
              name="Preferences"
              component={PreferencesScreen}
              options={{ title: fr.nav.preferences }}
            />
            <Stack.Screen
              name="Privacy"
              component={PrivacyScreen}
              options={{ title: fr.nav.privacy }}
            />
            <Stack.Screen
              name="BlockedUsers"
              component={BlockedUsersScreen}
              options={{ title: fr.nav.blocked }}
            />
            <Stack.Screen
              name="ChangePassword"
              component={ChangePasswordScreen}
              options={{ title: fr.nav.changePassword }}
            />
            <Stack.Screen
              name="DeleteAccount"
              component={DeleteAccountScreen}
              options={{ title: fr.nav.deleteAccount }}
            />
            <Stack.Screen
              name="Report"
              component={ReportScreen}
              options={{ title: fr.nav.report, presentation: 'modal' }}
            />
          </Stack.Navigator>
        </RealtimeProvider>
      </MatchProvider>
    </DeckProvider>
  );
}
