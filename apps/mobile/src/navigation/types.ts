import type { NavigatorScreenParams } from "@react-navigation/native";
import type { PublicProfile, ReportReason } from "../api/types";

export type AuthStackParams = {
  Welcome: undefined;
  Login: undefined;
  Register: undefined;
  ForgotPassword: { email?: string } | undefined;
};

export type TabParams = {
  Discover: undefined;
  Likes: undefined;
  Messages: undefined;
  Me: undefined;
};

export type MainStackParams = {
  Tabs: NavigatorScreenParams<TabParams> | undefined;
  Chat: { conversationId: string; name?: string };
  ProfileDetail: {
    userId: string;
    profile?: PublicProfile;
    canSwipe?: boolean;
    matchId?: string;
    conversationId?: string;
  };
  EditProfile: undefined;
  Photos: undefined;
  Preferences: undefined;
  Notifications: undefined;
  Settings: undefined;
  BlockedUsers: undefined;
  Report: { userId: string; name: string; defaultReason?: ReportReason };
};
