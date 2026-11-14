import type { NavigatorScreenParams } from "@react-navigation/native";
import type { PublicProfile } from "../api/types";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
  ResetPassword: { email: string };
};

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Me: undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Conversation: { matchId: string; user: PublicProfile };
  ProfileDetail: { userId: string; profile?: PublicProfile; matchId?: string };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  BlockedUsers: undefined;
  DeleteAccount: undefined;
};

export type RootParamList = AuthStackParamList & MainStackParamList & TabParamList & { Onboarding: undefined };

declare global {
  namespace ReactNavigation {
    interface RootParamList extends MainStackParamList, AuthStackParamList {}
  }
}
