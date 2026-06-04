export interface ProjectCardModel {
  title: string;
  desc: string;
  status: string;
  genre: string;
  followers: string;
}

export interface FeedCardModel {
  author: string;
  title: string;
  desc: string;
  meta: string;
}

export interface RecruitCardModel {
  project: string;
  role: string;
  mode: string;
  days: string;
}
