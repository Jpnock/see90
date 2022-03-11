int f(int y, int z)
{
    int a = 10;
    int *b = &a;
    int **c = &b;
    int ***d = &c;
    int **e = *d;
    int f = 3;
    **e += 2;
    f += 1;
    return f + **e + 5;
}

/*
    int *my_func(x,y )

    int *a = malloc(sizeof(int) * 10);
    a[2] = 5;

    *(a+2) = 5;

    my_func(1, 3)[5] = 5;


    int c = 5;
    int *b = &c;
    int **a = &b;
    *a[5] = 6;
*/