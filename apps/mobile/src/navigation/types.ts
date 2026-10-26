import type { NavigatorScreenParams } from "@react-navigation/native";
import type { PublicProfile } from "../api";

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Activity: undefined;
  Me: undefined;
};

export type RootStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList>;
  Chat: { matchId: string; profile: PublicProfile };
  ProfileDetail: { profile: PublicProfile; matchId?: string; canSwipe?: boolean };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  Blocked: undefined;
  Account: undefined;
};

export type AuthStackParamList = {
  Welcome: undefined;
  Forgot: { email?: string } | undefined;
};

declare global {
  namespace ReactNavigation {
    // eslint-disable-next-line @typescript-eslint/no-empty-object-type
    interface RootParamList extends RootStackParamList {}
  }
}
