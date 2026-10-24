import type { NavigatorScreenParams } from "@react-navigation/native";
import type { MatchInfo } from "../api/types";

export type AuthStackParamList = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type TabParamList = {
  Discover: undefined;
  Messages: undefined;
  Profile: undefined;
};

export type MainStackParamList = {
  Tabs: NavigatorScreenParams<TabParamList> | undefined;
  ProfileDetail: { userId: string; canSwipe?: boolean };
  Chat: { conversationId: string; userId: string; name: string; photoId?: string };
  Notifications: undefined;
  EditProfile: undefined;
  ManagePhotos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  BlockedUsers: undefined;
  DeleteAccount: undefined;
  MatchFound: { match: MatchInfo };
};
