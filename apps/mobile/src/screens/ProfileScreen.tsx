import { createNativeStackNavigator } from "@react-navigation/native-stack";
import React from "react";
import { strings } from "../i18n/strings";
import type { ProfileStackParamList } from "../navigation/types";
import { useStackScreenOptions } from "../navigation/theme";
import BlockedUsersScreen from "./profile/BlockedUsersScreen";
import DeleteAccountScreen from "./profile/DeleteAccountScreen";
import EditProfileScreen from "./profile/EditProfileScreen";
import PhotosScreen from "./profile/PhotosScreen";
import PreferencesScreen from "./profile/PreferencesScreen";
import ProfileHomeScreen from "./profile/ProfileHomeScreen";

const Stack = createNativeStackNavigator<ProfileStackParamList>();

/** Profile tab: a small nested stack (profile -> edit / photos / preferences / blocked / delete). */
export default function ProfileScreen() {
  const screenOptions = useStackScreenOptions();
  return (
    <Stack.Navigator screenOptions={screenOptions}>
      <Stack.Screen component={ProfileHomeScreen} name="ProfileHome" options={{ headerShown: false }} />
      <Stack.Screen
        component={EditProfileScreen}
        name="EditProfile"
        options={{ title: strings.profile.edit }}
      />
      <Stack.Screen component={PhotosScreen} name="Photos" options={{ title: strings.profile.photos }} />
      <Stack.Screen
        component={PreferencesScreen}
        name="Preferences"
        options={{ title: strings.profile.preferences }}
      />
      <Stack.Screen
        component={BlockedUsersScreen}
        name="BlockedUsers"
        options={{ title: strings.profile.blocked }}
      />
      <Stack.Screen
        component={DeleteAccountScreen}
        name="DeleteAccount"
        options={{ title: strings.profile.deleteAccount }}
      />
    </Stack.Navigator>
  );
}
