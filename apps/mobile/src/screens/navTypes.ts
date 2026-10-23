import type { NavigatorScreenParams } from "@react-navigation/native";
import type { Card } from "../api/types";

export type AuthStackParams = {
  Login: undefined;
  Register: undefined;
  ForgotPassword: undefined;
  ResetPassword: { email: string };
};

export type TabParams = {
  Discover: undefined;
  Matches: undefined;
  Notifications: undefined;
  Profile: undefined;
};

export type ChatParams = {
  matchId: string;
  userId?: string;
  firstName?: string;
  thumbUrl?: string | null;
};

export type MainStackParams = {
  Tabs: NavigatorScreenParams<TabParams> | undefined;
  Chat: ChatParams;
  ProfileDetail: { userId: string; card?: Card; canSwipe?: boolean };
  EditProfile: undefined;
  ManagePhotos: undefined;
  Preferences: undefined;
  Privacy: undefined;
  Blocked: undefined;
  DeleteAccount: undefined;
};
