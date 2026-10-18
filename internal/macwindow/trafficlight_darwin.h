#ifndef BUCKY_TRAFFICLIGHT_DARWIN_H
#define BUCKY_TRAFFICLIGHT_DARWIN_H

// Moves the standard window buttons so the cluster starts left points from the
// window's left edge and its vertical centre sits centreY points below the top.
// Repeated calls update the target rather than adding observers. Any thread.
void BUCKYSetTrafficLightPosition(void *nsWindow, double left, double centreY);

#endif
