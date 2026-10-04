import type { NavigatorScreenParams } from "@react-navigation/native";
import type { MatchSummary } from "../api/types";

export type AuthStackParamList = {
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
  ResetPassword: { email?: string } | undefined;
};

export type TabParamList = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  Chat: { match: MatchSummary };
  ProfileDetail: { userId: string; source?: "discover" | "match" };
  EditProfile: undefined;
  ManagePhotos: undefined;
  Preferences: undefined;
  BlockedUsers: undefined;
  ChangePassword: undefined;
  DeleteAccount: undefined;
  SafetyTips: undefined;
};

declare global {
  namespace ReactNavigation {
    interface RootParamList extends MainStackParamList, AuthStackParamList {}
  }
}
