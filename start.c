/*************************************************************************
	> File Name: start.c
	> Author: Logan Von
	> Mail: loganwhitevon@gmail.com
	> Created Time: Mon 05 Jun 2017 05:07:03 PM CST
 ************************************************************************/

#include <unistd.h>
#include <stdio.h>

int main() {
    int ret, child_ret, status;    
    pid_t pid;

	if ((pid = fork()) < 0) {
        printf("Error while forking!\n");
        _exit(-1);
    } else if (pid == 0) {      /* child process */
        if (execl("./main", "main", (char*)0) < 0) {
            printf("Error while remounting /!\n");
            _exit(-1);
        }
    } else {
        return 0;
    }
}
